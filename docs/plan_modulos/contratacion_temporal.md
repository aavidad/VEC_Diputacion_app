# Contratación temporal

**Estado a 8 de octubre de 2026:** en cidonia RRHH tramita un expediente desde la petición del centro hasta la resolución, con el alta según la circular de mayo (CT193, instalada con HZ8); faltan el recorrido completo con reinicio hasta el cese, el paso a Personal, la incorporación acreditada y la firma de dos personas.

Base comprobada: `origin/main@033fda6ed` más las PR fusionadas el 08/10 (#895 a #904). Los criterios para dar CT por cerrada están en [OBJETIVOS.md](OBJETIVOS.md). La firma tiene su plan en [firmas.md](firmas.md). El circuito que describió RRHH está en [decisiones_rrhh_2026-10-02.md](../estudio_requisitos/decisiones_rrhh_2026-10-02.md).

## Lo que falta

| # | Qué | Quién o qué lo frena | Cómo se comprueba |
| --- | --- | --- | --- |
| 1 | **Recorrido completo con el main actual**: petición, ratificación, análisis, llamamiento, aceptación, informe, fiscalización con un reparo, resolución, GINPIX, Personal, cese y vuelta a Bolsa; después, reinicio de aplicación y PostgreSQL. El único acta ([05/10](../estudio_requisitos/acta_recorrido_ct_bolsa_20261005.md)) se paró en la resolución de la aceptación; sus fallos están arreglados salvo ORQ-1. | Claude. Necesita antes los puntos 2 y 3. | Acta nueva con respuesta HTTP y captura por paso, antes y después del reinicio. |
| 2 | **ORQ-1: selección a medias entre CT y Bolsa.** Si la selección falla entre los dos módulos, el expediente queda bloqueado. Es lo único del acta sin arreglo. | Sin dueño. Hay que encargarlo. | Fallo provocado a mitad de la selección y el expediente se recupera con la misma clave. |
| 3 | **Encender en cidonia lo que ya existe**: incorporación acreditada y cancelación por el centro (`VEC_CT_INCORPORACION_ACREDITADA_ENABLED=false`), paso a Personal (`VEC_PERSONAL_B2_GOBIERNO_ENABLED=false`, con su SQL de Personal pendiente: AD211/P22, AD175/P32, AD180/P34 y siguientes) y auditoría de RRHH (`VEC_RRHH_AUDITORIA_ENABLED`, ausente). | Claude, en el despliegue. Duda 140 para el momento de la toma de posesión. | Los pasos 17 y siguientes del acta responden 200, no 404. |
| 4 | **PR abiertas de CT.** #910 ficha y llamamiento visibles cuando la categoría no tiene bolsa (Codex-S); #906 nombres de centro en la lista ligera (Codex-S); #905 peticiones del centro sin consultas duplicadas (equipo W, en conflicto); #907 CT194, validar el alta en SQL (Codex-X); #540 auditoría de consultas fallidas (Codex-T). | Sus equipos. Revisión y fusión: Claude. | CI verde, revisiones y fusión. |
| 5 | **Plazos de CT editables en pantalla.** Hoy los plazos de CT solo se leen. | Codex-Y, rama `codexy-plazos-20261008` sin PR (CT190, 191, 195 y 196 reservadas). Al fusionarla se cierran #233, #237 y #240. | RRHH cambia un plazo, queda versión y auditoría, y el cuadro lo usa. |
| 6 | **Tarjetas de plazo en Inicio (CT192).** | Codex-S, rama `codexs-ct-filtros-plazos-20261008` sin PR. | Cada tarjeta abre su lista filtrada. |
| 7 | **Cuadro de CT por debajo de 300 ms.** La última medida (#840) dio 363–372 ms de p95. | Sin dueño. Medir primero en cidonia. | p95 menor de 300 ms con el volumen de la principal. |
| 8 | **Documentos de la ficha en un solo sitio.** | Codex-S. Hay un commit `046a2f2fa` solo en local, sin subir. | Todos los documentos del expediente se ven y descargan desde la ficha. |
| 9 | **Ruta muerta del cierre administrativo.** `RutaCerrarAdministrativamente` y `RutaReabrirExcepcionalmente` responden siempre «no disponible»; la pantalla ya usa `/seguimiento/cerrar-sin-cese`. Retirarlas. | Sin dueño. Tarea pequeña. | Ninguna ruta de CT responde «no disponible» sin motivo. |
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

## Edición de plazos CT: preparación del 8 de octubre

La rama `trabajo/codexy-ct-plazos-minimo-20261008` recupera la edición de plazos en la pantalla «Reglas vigentes». Conserva CT190 y CT191: la primera guarda la base y los ajustes al abrir una fase; la segunda comprueba la base activa al guardar un cambio. La lista `deploy/principal/lista_sql_codexy_ct_plazos_minimo_20261008.txt` incluye únicamente esas dos migraciones y sus SHA256. CT148 y AD114 siguen siendo la autoridad de las operaciones; no se recrean sus tablas ni permisos. CT195, AD223, B88, B89 y el correo quedan fuera de esta rama.

La interfaz ofrece revisión antes de guardar, motivo, conflicto sin perder el borrador, recibo e historial paginado. Carga solo su catálogo en el idioma activo y permite reintentar si falla. El ensayo PostgreSQL 18 reprodujo postHX, globals, los catorce cambios HZ, B85, B86, CT193 y B87. Conservó los 170 tramos anteriores y las cinco definiciones de CT164. Las pruebas de CT191 emplean un doble V3 dentro de ROLLBACK; no prueban un acto autorizado real. Chrome comprobó la interfaz con respuestas de prueba, en castellano e inglés, escritorio, 390 px y ampliación del 200 %.

El recorrido sigue pendiente. Main no ofrece al módulo el proveedor común de sesión en memoria y auditoría de lecturas de éxito que exige la decisión del 7 de octubre. El repositorio cierra GET y la prelectura de POST mientras falta ese lector. Los paquetes no están montados en el servidor, y el cuadro no consume todavía la captura persistida. No habilitar ni instalar esta preparación como edición disponible. La siguiente tarea consiste en recibir la fuente de Identidad y Auditoría, componerla con los perfiles nominales existentes y verificar navegador, recibo y recuperación tras reinicio. Las PR #233, #237 y #240 podrán cerrarse cuando se fusione esa sustitución completa. Claude conserva la revisión SQL y la fusión.
