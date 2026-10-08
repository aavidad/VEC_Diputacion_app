# Contratación temporal

**Estado a 8 de octubre de 2026:** en cidonia RRHH tramita un expediente desde la petición del centro hasta la resolución, con el alta según la circular de mayo (CT193, instalada con HZ8); faltan el recorrido completo con reinicio hasta el cese, el paso a Personal, la incorporación acreditada y la firma de dos personas.

Base comprobada: `origin/main@033fda6ed` más las PR fusionadas el 08/10 (#895 a #904). Los criterios para dar CT por cerrada están en [OBJETIVOS.md](OBJETIVOS.md). La firma tiene su plan en [firmas.md](firmas.md). El circuito que describió RRHH está en [decisiones_rrhh_2026-10-02.md](../estudio_requisitos/decisiones_rrhh_2026-10-02.md).

## Lo que falta

| # | Qué | Quién o qué lo frena | Cómo se comprueba |
| --- | --- | --- | --- |
| 1 | **Recorrido completo con el main actual**: petición, ratificación, análisis, llamamiento, aceptación, informe, fiscalización con un reparo, resolución, GINPIX, Personal, cese y vuelta a Bolsa; después, reinicio de aplicación y PostgreSQL. El único acta ([05/10](../estudio_requisitos/acta_recorrido_ct_bolsa_20261005.md)) se paró en la resolución de la aceptación; sus fallos están arreglados salvo ORQ-1. | Claude. Necesita antes los puntos 2 y 3. | Acta nueva con respuesta HTTP y captura por paso, antes y después del reinicio. |
| 2 | **ORQ-1: selección a medias entre CT y Bolsa.** Recuperación de las dos ventanas comprobada en clones PostgreSQL 18 con autorización nominal, misma clave y reinicio. Candidato #918, aún pendiente de integración. | Codex-X entrega; Claude revisa y fusiona. | Recibo y fecha originales; replays Chrome 200/200 sin duplicar Bolsa. |
| 3 | **Encender en cidonia lo que ya existe**: incorporación acreditada y cancelación por el centro (`VEC_CT_INCORPORACION_ACREDITADA_ENABLED=false`), paso a Personal (`VEC_PERSONAL_B2_GOBIERNO_ENABLED=false`, con su SQL de Personal pendiente: AD211/P22, AD175/P32, AD180/P34 y siguientes) y auditoría de RRHH (`VEC_RRHH_AUDITORIA_ENABLED`, ausente). | Claude, en el despliegue. Duda 140 para el momento de la toma de posesión. | Los pasos 17 y siguientes del acta responden 200, no 404. |
| 4 | **PR abiertas de CT.** #910 ficha y llamamiento visibles cuando la categoría no tiene bolsa (Codex-S); #906 nombres de centro en la lista ligera (Codex-S); #905 peticiones del centro sin consultas duplicadas (equipo W, en conflicto); #907 CT194, validar el alta en SQL (Codex-X); #540 auditoría de consultas fallidas (Codex-T). | Sus equipos. Revisión y fusión: Claude. | CI verde, revisiones y fusión. |
| 5 | **Plazos de CT editables en pantalla.** Hoy los plazos de CT solo se leen. | Codex-Y, PR #919, sólo CT190 y CT191. Ensayo y correcciones de la revisión en curso; Claude revisa las SQL y fusiona. Al fusionarla se cierran #233, #237 y #240. | RRHH cambia un plazo, queda versión y auditoría, y el cuadro lo usa. |
| 6 | **Tarjetas de plazo en Inicio (CT192).** | Codex-S, rama `codexs-ct-filtros-plazos-20261008` sin PR. | Cada tarjeta abre su lista filtrada. |
| 7 | **Cuadro de CT por debajo de 300 ms.** #916 fusionada: elimina las relecturas por fila si falla la preparación de reglas. Medida local nominal: p95 HTTPS 208,978 ms con 5.071 expedientes; no se atribuye esa cifra al parche Go ni a cidonia. | Codex-X; medición y límites en el acta de rendimiento. | 27 consultas con límites 1 y 100; auditoría por petición, sin N+1. |
| 8 | **Documentos de la ficha en un solo sitio.** | Codex-S. Hay un commit `046a2f2fa` solo en local, sin subir. | Todos los documentos del expediente se ven y descargan desde la ficha. |
| 9 | **Retirada de dos rutas inoperativas de cierre.** #915 elimina `RutaCerrarAdministrativamente` y `RutaReabrirExcepcionalmente`, sus dependencias vacías y entradas de inventario. La pantalla conserva `/seguimiento/cerrar-sin-cese` y su preparación; `/cierres-expediente` mantiene su autoridad. | Codex-V: candidato revisado y puerta local verde; pendiente CI de integración y fusión de Claude. | Las dos URL dejan de despacharse. No se implementa reapertura ni se equipara al cierre sin cese; el rechazo genérico depende del perfil y la identidad. |
| 10 | **Bandeja de Intervención.** No consta una bandeja propia; el formulario de fiscalización se abre desde la ficha. | Comprobar con RRHH si hace falta. | — |
| 11 | **Firma de dos personas.** Ver [firmas.md](firmas.md). | Codex-V. | — |

## Esperan a RRHH

No se programan hasta tener respuesta. Las propuestas con fuente pública están en la PR #908.

- Plantillas Word de contratos, informes y resoluciones (duda 124). La herramienta de plantillas ya existe.
- Oferta al SAE: datos, canal y selección (dudas 72, 73 y 126). Reunión con RRHH.
- Delegaciones de firma y suplencias (dudas 122, 128 y 143), con Secretaría General.
- Portafirmas Firmadoc (duda 74), con Informática.
- Qué cuenta como toma de posesión y fecha de efectos del cese (duda 140).
- Quién declara la urgencia y en qué documento (duda 63).
- Coste cuando no hay fecha de fin (duda 146).
- Plazo de fiscalización tras subsanar (duda 95): el RD 424/2017 (arts. 10.2 y 12.4) manda un cómputo nuevo, que es lo que VEC ya hace. Se puede retirar de `dudas.md`.

## Ya hecho, no reencargar

- Alta según la circular de mayo con número de personas y puesto RPT exacto: #895, #897, #899 y #900 (CT193).
- Ficha sin pedir borradores no montados: #901. RRHH entra en peticiones del centro sin «Acceso denegado»: #904.
- Selectores con aviso al arrancar: #745. Sin crédito no se ofrece: #766. Coste por partidas: #775.
- Arreglos del acta del 05/10: CT179 y CT180 (`21f7bc496`), #767, #768, #771, #777, #778, #808 y #809.
- Arreglos de usabilidad: #750, #754–#757, #761, #762, #878 y #884.
- Nombre del centro en la ficha: #763. Documentos del expediente en la custodia: #845 y #850.
- Listas más rápidas (CT187): #840. Acceso al SAE retirado de Inicio hasta que tenga gestión: #893.

Ficheros que conviene no engordar: `internal/app/bootstrap/contratacion_temporal_desarrollo.go`, `contratacion_temporal_postgresql_desarrollo.go` y `contratacion_temporal_seguimiento_cese_desarrollo.go`. Lo nuevo va en ficheros nuevos.

## Edición de plazos CT desde «Reglas vigentes» — 8 de octubre

La rama `trabajo/codexy-ct-plazos-minimo-20261008` conecta la edición de plazos en la pantalla existente. RRHH revisa el cambio, elige su motivo y guarda una nueva versión, con recibo e historial paginado. Los textos están en castellano e inglés y se cargan sólo al abrir la pantalla, en el idioma activo. Los errores permiten reintentar; un conflicto conserva el borrador.

La lectura común de reglas permanece intacta. La prelectura comparte su resolutor; el guardado y la consulta del historial reutilizan AD114 y CT148. La ruta usa el perfil fijo `entrega-peticion-rrhh-lector`, cuyo ámbito de organización coincide con el recurso de ajustes. La ampliación de permisos exige la provisión administrativa existente, aprobada por huella y CAS. No se crea otro perfil ni se conceden permisos desde HTTP.

Se conservan únicamente CT190 y CT191, en `deploy/principal/lista_sql_codexy_ct_plazos_minimo_20261008.txt` con sus SHA256. CT190 conserva las reglas al abrir cada fase y añade los metadatos a las consultas existentes v4/v5, sin cambiar el canon ni el recibo del cuadro. Sus agregados distinguen las capturas por claves, sin copiar los catálogos en cada grupo. La v5 recupera las definiciones por identificador, versión y huella distintos y transporta cada definición una sola vez. La primera activación fija una referencia de transición para los tramos anteriores; no les atribuye una regla histórica ni modifica sus filas. Exige cero ajustes previos y la misma base que usa Go. Mientras no haya base activa, las altas y los cambios de fase continúan como legado sin captura, con el cálculo actual preparado una sola vez por consulta. Tras desactivar la base, la referencia de transición sólo se aplica a fases iniciadas hasta la primera activación. Página y resumen comparten la validación de los catálogos capturados por sus dos huellas; cada tramo conserva la comprobación de bytes, metadatos, fase y fecha. El despliegue instala primero las SQL y después el binario. CT191 comprueba la base activa al guardar. CT195, AD223, B88, B89 y el correo quedan fuera.

CONFIG NUEVA: `VEC_CT_REGLAS_AJUSTES_ENABLED=true` y `VEC_CT_REGLAS_AJUSTES_MOTIVOS_PATH=/vec-incorporacion/motivos_ajuste_ct_v1.json`. Se copia allí `data/catalogos/contratacion_temporal/motivos_ajuste_v1.json`. La fuente CT existente debe coincidir con la base publicada y activa. La secuencia de publicación y provisión está en `cmd/vec-preparar-catalogo-reglas-ct/README.md`; las variables de aprobación CAS se retiran después de usar la preimagen exacta.

La preparación previa pasó la puerta completa y Chrome con respuestas de prueba a 1440 y 390 px y zoom del 200 %. Las pruebas focales del montaje y las capturas pasaron. El ensayo reversible de CT190 comprueba legado, fase nueva, cambio posterior y lectores v4/v5; emplea un doble V3 dentro de ROLLBACK y no acredita un acto nominal. El cierre de cadena, la calidad final y la revisión de los bytes finales se registran en la PR. La instalación y el recorrido nominal en cidonia corresponden a Claude. Esta entrega sustituye #233, #237 y #240 cuando se fusione; no se instalan sus CT157/CT158 antiguas.

La corrección del dictamen SQL de la #919 se ensayó de nuevo sobre postHX, HZ14, B85, B86, CT193, B87, CT194, CT197, B81 y B90. La prueba transaccional conserva las 170 publicaciones y fases; pasa el alta inicial, la herencia de fase y el cambio de fase sin base activa, además de una base desactivada. Los cinco cuerpos CT164 permanecen idénticos. Con 5.170 publicaciones y una base de 26.088 bytes, el p95 de diez mediciones SQL es de 67,23 ms para el resumen y 204,50 ms para v5; las capturas de página y grupos coinciden con la salida anterior. La carga de volumen omite las claves externas del resto del circuito y la medición usa un doble transaccional de V3; no es una medida HTTP ni un acto nominal. La revisión SQL final y la fusión corresponden a Claude.

La segunda corrección se ensayó sobre esa misma cadena más B94. CT190 comprueba las huellas de los cuerpos instalados antes de bloquear tablas y conserva los comentarios de CT184. La transición sólo cubre fases iniciadas hasta la primera activación. El cálculo comparte por consulta los catálogos validados y sus canónicos; una fase sin plazo también reutiliza el catálogo, y las salidas públicas conservan copias independientes. Con 5.000 expedientes capturados añadidos, 5.170 publicaciones y 5.071 grupos, 30 POST del manejador y servicio reales dieron p95 de 282,852 ms, frente a 311,391 ms antes del último ajuste. Hubo una consulta v5 por petición y ninguna consulta SQL por grupo. El arnés comprueba una fila de página con límite 100 y todo el resumen; identidad, emisor, recibo y calendario son de prueba. La medida incluye el cálculo CT/Calendarios y v5, pero no TLS ni autorización nominal. Los 170 registros anteriores y cinco cuerpos CT164 permanecen intactos en el ensayo.

## Validación de nuevas peticiones — 8 de octubre de 2026

CT194 añade comprobaciones SQL del porcentaje, los códigos, las referencias, las fechas y los textos de la necesidad. Conserva los errores de concurrencia para que el alta pueda reintentar la transacción; una dependencia no disponible sigue devolviendo el error nominal. La migración parte de las dos funciones instaladas después de postHX, los 14 cambios HZ, B85, B86, CT193 y B87. No cambia las 71 altas conservadas ni sus recibos, historia, permisos o circuito inicial CT164.

La publicación vigente del catálogo de necesidades es v3 y cita la circular de peticiones de personal firmada el 8 de mayo de 2026. Conserva la misma ruta configurable. Los expedientes anteriores mantienen su instantánea y huella, incluida la publicación v2 que citaba la circular de febrero. El GET del catálogo conserva el esquema v2. El alta limitada sin catálogo ya está en main y no se vuelve a implementar. Para desplegar sólo se instala CT194 una vez, según `deploy/principal/lista_sql_codexx_p2_alta_20261008.txt`; no requiere configuración nueva. El ensayo PostgreSQL 18 y las pruebas de reintento SQL y respuesta HTTP pasan. La medición de la validadora dio 0,099 ms por llamada en 1000 llamadas locales; no mide una petición HTTP completa. Las tarjetas de plazo CT192 siguen en una pieza separada y necesitan la sesión nominal y el apunte común en la misma transacción.


## Ficha después de registrar el análisis — 8 de octubre de 2026

La corrección conserva el recibo de análisis, libera la navegación después de confirmar y actualiza la ficha mediante una única lectura del detalle autorizado a la versión del recibo. Mantiene filtro y página aunque el expediente cambie de fase o salga de esa lista. La respuesta tardía no sustituye otra selección; una lectura fallida conserva el recibo y muestra el aviso pendiente sin repetir el registro. El replay HTTP 200 acepta el mismo recibo validado que 201. Al volver al cuadro se limpia también la URL de ficha; una carga anterior que llegue tarde conserva la selección posterior.

Las pruebas reprodujeron los bloqueos antes de corregirlos. La revisión independiente con Chrome detectó además pérdida de foco, aviso sobrescrito y URL de JavaScript antiguas; se corrigieron antes de la entrega. La cohorte `20261008-alta-analisis-bolsa-fichas-v5` renueva los importadores hasta la entrada y la caché del portal, conservando Alta final de #924 y Bolsa global. Los tamaños registrados corresponden a los archivos finales de la integración. No añade SQL, perfiles ni configuración. La verificación local de interfaz usa contratos de prueba; R repetirá el recorrido nominal en cidonia tras la publicación de Claude.

## ORQ-1: recuperación de la selección pendiente — 8 de octubre de 2026

La recuperación conserva la clave y la intención originales. CT vuelve a autorizar la ventana pendiente; Bolsa devuelve su orden o llamamiento original. El recibo recuperado mantiene referencia y fecha y lleva su marca autenticada. Una reserva antigua no puede confirmar después de renovarse.

La lista de instalación es `deploy/principal/lista_sql_codexx_orq1_20261008.txt`: AD225 → CT198 → AD226 → CT199, una sola vez y después de IS17, CA37, AD215, AUT57, AD211, P22, AD175, P32, AD180, P34, AD214, AD216 y AD218. AD225 añade sólo el bloque de recuperación sobre la definición instalada exacta posterior a AD218; conserva ACL, configuración y el resto del núcleo. No sustituye el núcleo completo ni permite invertir la cadena anterior. AD226 registra el origen técnico de la nueva acción copiando LOGIN, audiencia, canal y proceso de la recuperación de orden ya configurada; no concede perfiles ni crea roles. CT199 completa el material de huella que faltaba para el recibo recuperado. Las dos SQL finales se ensayaron desde ausencia del origen en una base causal nueva; las seis selecciones anteriores conservan sus huellas. Se requiere el sellador de auditoría existente activo. No hay variable de entorno nueva.

En dos clones privados se provocó el fallo antes de guardar en Bolsa y después de guardar la propuesta pero antes de confirmar CT. Tras restituir el permiso original de ensayo y reiniciar aplicación/PostgreSQL, la misma petición recuperó el expediente. En la segunda ventana, dos replays simultáneos desde Chrome devolvieron 200/200 y exactamente el recibo, fecha y versión previos. Bolsa conservó 19 integraciones, 19 asientos de negocio y 19 eventos; los reintentos dejaron sus apuntes nominales sin duplicar el efecto. Versión alterada: 409; certificado de Intervención: 403 con apunte de denegación.

La comprobación de Chrome ejercitó la API con el certificado sintético del kit y un contexto aislado que acepta el certificado servidor local; curl sí verificó ese certificado. No fue un recorrido del formulario. Las revisiones independientes y pruebas focales SQL/Go pasaron; quedan la CI de integración y la fusión de Claude. No se atribuyen a este ensayo la revocación CAS ni el trámite posterior a selección. Las bases y servicios propios de prueba se retiraron.


Revisión SQL de instalación: el primer candidato ORQ fue rechazado por excluir la cadena Personal/B1. La versión final usa un parche localizado con marca única y exige exclusivamente la preimagen posterior a AD218. Antes de esa cadena se detiene sin crear la fachada ni cambiar metadatos o historia; el ensayo comparó ambos estados completos. En un PostgreSQL 18 nuevo se instalaron los 13 prerrequisitos antes de las cuatro SQL de ORQ; las pruebas pasaron. Tras fallo persistido en solicitar llamamiento y reinicio, dos recuperaciones simultáneas reales devolvieron 200/503 por la reserva; el replay final200 fue idéntico al ganador y al recibo/fecha previos de Bolsa, con una sola historia de recuperación y19/19/19 de negocio. El arnés y las pruebas están en pruebas_sql; no se reaplicaron SQL confirmadas.
