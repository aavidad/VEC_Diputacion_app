# O3-02 — caso de uso saneado de análisis RRHH

Fecha: 23 de julio de 2026.

Estado: integrado en `e9d461c` tras corregir tres `NO-GO`, obtener revisión
independiente con GO y superar las puertas del árbol conjunto. O3-02 está
cerrada técnicamente. No habilita datos reales ni acredita producción, ENS,
ENI o conformidad jurídica; faltan O3-04, O3-05 y las aprobaciones formales.

## Alcance

Este corte implementa la coordinación de aplicación para registrar y
rectificar el análisis de un expediente de contratación temporal. Conserva la
arquitectura hexagonal y no introduce HTTP, cookies, SQL, SDK de proveedor ni
almacenamiento del navegador.

Incluye:

- comandos neutrales para web, escritorio, CLI y MCP;
- contexto de identidad de garantía alta resuelto por la frontera VEC;
- artefacto interno opaco acuñado desde pruebas O3-03 verificadas, pero sin
  consumirlas ni producir efectos durables;
- recuperación idempotente temprana de una confirmación exacta, incluso si las
  fuentes O3-03 ya no están disponibles;
- orden indivisible pendiente para consumir conjuntamente RC y coste;
- idempotencia semántica con HMAC rotatorio;
- política gobernada, versionada y ligada al estado previo;
- autorización VEC V3 exacta;
- control optimista de versión y rectificación motivada;
- segregación obligatoria entre quien registró y quien rectifica;
- orden opaca que transporta contexto, evidencia, consumo pendiente y
  concesión V3 a una única transacción final;
- recibo verificable, cancelación y errores públicos sin causas privadas.

No incluye:

- adaptador PostgreSQL de preparación o confirmación;
- adaptadores reales de RC, coste, verificación TCB, catálogo, credenciales ni
  consumo durable;
- adaptador durable O3-04 que, en un único `COMMIT`, consume conjuntamente
  RC+coste y V3 y confirma agregado, historia, auditoría, recibo y outbox;
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

`CapacidadPrepararArtefactoAnalisisO3`, situada en `application`, obtiene las
peticiones selladas desde infraestructura interna, invoca los puertos O3-03,
contrasta credenciales y pruebas de posesión contra la confianza fijada por la
composición, verifica respuesta, TCB y catálogo, presenta un desafío nuevo a
cada autoridad y acuña un artefacto verificado todavía sin consumo. En esta
revalidación O3-02, `application` coordina todas las presentaciones y cruces.
`ports` sólo aporta contratos, DTO opacos, copias defensivas y validación
criptográfica local neutral; no invoca ni coordina fuente, verificador y
publicador.

La confirmación de cada nuevo desafío es nominal y opaca: sólo puede nacer
después de verificar credencial y prueba de posesión, y liga material, rol,
instante e identidad. Una proyección que copie únicamente el vínculo público
no puede simular la revalidación.

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
- orden exacta de consumo conjunto todavía pendiente, ligada por huella a RC
  y coste;
- instante de preparación y huella SHA-256 determinista del contenido.

Esta huella es una dirección de contenido, no una segunda autoridad
criptográfica. La autoridad procede de O3-03. Los motivos negativos de RC
conservan por separado la entrada del catálogo y la clave i18n; no se exige
que sean el mismo identificador ni se aceptan textos libres como autoridad.
El artefacto nunca incorpora un recibo de consumo. El recibo sólo puede nacer
dentro de la transacción final y queda ligado a la huella de la orden pendiente
y a la referencia de la concesión V3. No se acepta desde el cliente ni actúa
como autoridad independiente.

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
6. Desde `application`, revalidar las autoridades con desafíos nuevos. Cada
   comprobación opaca liga material, rol, instante e identidad; después se
   exige coincidencia exacta con la evidencia original y se acuña el artefacto
   sin recibos de consumo.
7. Construir y sellar las preimágenes de ámbito y semántica.
8. Reservar o recuperar la intención idempotente. Una recuperación confirmada
   en esta segunda barrera tampoco repite el consumo.
9. Resolver y validar la política gobernada para fase y estado previos.
10. Exigir siempre actor distinto en una rectificación.
11. Obtener y revalidar la concesión VEC V3 ligada al recurso exacto.
12. Extraer del artefacto la `OrdenConsumoConjuntoFuentesAnalisisO3` exacta,
    todavía pendiente y sin producir ningún efecto.
13. Reproducir `RegistrarAnalisis` o `RectificarAnalisis` sobre una copia del
    agregado anterior.
14. Construir una orden final que vuelve a ejecutar la transición, exige
    igualdad completa del expediente resultante y transporta el contexto, la
    orden pendiente de fuentes y la concesión V3 exacta.
15. Revalidar en aplicación todo lo comprobable antes de entrar en la frontera
    durable.
16. En O3-04, adquirir bloqueos y revalidar con el reloj autoritativo de la
    base de datos contexto, evidencia, CAS, política, idempotencia y V3.
17. En esa misma transacción consumir conjuntamente RC+coste y V3, persistir
    agregado, historia, auditoría, recibo y outbox, validar el recibo y sólo
    entonces ejecutar un único `COMMIT`.
18. Al volver, validar el recibo antes de procesar cualquier error simultáneo.
    Un recibo válido y concordante demuestra el `COMMIT` aunque coincida con
    cancelación o error de transporte. Un recibo vacío, adulterado o ajeno
    nunca prevalece. No se lee un reloj local nuevo ni se reinterpreta el
    instante autoritativo declarado por el propio recibo.

No existe ya un puerto de preconsumo ni una compensación entre dos commits.
Una caída en aplicación, construcción de orden, CAS, persistencia, tiempo o
validación interna del recibo previa al `COMMIT` debe dejar RC, coste y V3 sin
consumir. Si un adaptador comete el defecto de alterar sólo la proyección que
devuelve después del `COMMIT`, aplicación no la expone y el replay exacto
recupera el recibo durable correcto. La misma referencia con otra huella
produce `ErrConjuntoFuentesAnalisisYaConsumido`.

O3-02 define y prueba el contrato atómico, pero no implementa todavía el
adaptador PostgreSQL O3-04. Hasta que ese adaptador supere pruebas reales de
caída, reinicio, concurrencia y resultado indeterminado, no se afirma
durabilidad ni preparación para producción.

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
  durante la preparación se rechaza antes de producir efectos.
- Una denegación de política, segregación o VEC V3 no consume RC ni coste.
- El replay exacto confirmado se recupera antes de consultar fuentes; las
  fuentes caídas no impiden devolver su recibo.
- El recibo conjunto no puede confirmar solo RC cuando la orden incluye coste;
  la orden final transporta exactamente la huella pendiente del artefacto.
- La fábrica de la orden reproduce la operación de dominio sobre el estado
  previo y compara el agregado completo, no una selección de campos.
- El instante del recibo final debe permanecer dentro de la vigencia del
  contexto, de las fuentes y de la concesión VEC V3, comprobado dentro de la
  transacción antes del `COMMIT`.
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
- Un recibo concordante gana frente a un error competitivo posterior al
  `COMMIT`; si el recibo no es confiable, se devuelve cero y se conserva la
  clasificación segura del error.

### Encapsulación

Artefacto, pruebas O3-03, preparación, orden y evidencia rechazan codificación
y decodificación JSON, XML, texto, binario, gob, CBOR y YAML. Esta barrera
evita que un adaptador reconstruya autoridad interna a partir de una carga
externa.

## Evidencia automatizada

Las pruebas del corte cubren:

- registro y rectificación correctos;
- integración real del caso de uso con peticiones, fuentes, credenciales,
  atestaciones, TCB, orden pendiente y derivación de RC y coste;
- ausencia del constructor nominal público y rechazo de una proyección
  nominal autoconsistente;
- revalidación de credenciales con desafío nuevo coordinada en `application`;
- confirmación opaca ligada a material, rol, instante e identidad, con rechazo
  de alias mutable, otro material u otra serie;
- adulteración de atestación, confirmación, credencial, catálogo y orden;
- replay exacto temprano, conflicto semántico y caducidad antes del commit;
- recuperación temprana con O3-03 caído;
- orden única para RC y coste, rechazo del recibo parcial y timestamp atómico
  común;
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
- recibo final fuera de la ventana de contexto, fuentes o concesión, con cero
  consumos y cero commits;
- fallo CAS y caída de persistencia con rollback total simulado;
- adaptador que altera el recibo de salida después del commit: el servicio no
  lo expone y el replay temprano recupera el recibo correcto sin repetir
  consumos;
- recibo válido junto con cancelación o error de transporte posterior al
  commit: prevalece el éxito probado; un recibo adulterado nunca gana;
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

1. O3-04: confirmación PostgreSQL atómica, con roles y ACL,
   concurrencia, reintento, reinicio y resultado indeterminado.
2. Composición real y, después, API, interfaz y E2E de O3-05.
