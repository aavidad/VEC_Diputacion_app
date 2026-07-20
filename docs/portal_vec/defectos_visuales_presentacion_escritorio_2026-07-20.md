# Resolución de defectos visuales de escritorio — presentación RRHH

Fecha de revisión y corrección: 20 de julio de 2026.

## Alcance

La revisión se realizó sobre la composición Docker real de presentación y las
capturas producidas por `scripts/capturar_presentacion_web.py`. El alcance
acordado para la presentación es escritorio: 1440×1000 y portátil: 1024×900.
La adaptación móvil a 390×844 se conserva en la prueba automática para evitar
regresiones graves, pero su pulido visual queda fuera de esta entrega por
decisión expresa de dirección.

La presentación usa exclusivamente datos sintéticos y documentación pública.
Las operaciones privadas son efímeras, no crean cookies ni almacenamiento de
navegador y no tienen efectos administrativos.

## Estado de los hallazgos

### H-01 — Directorio público de categorías

**Estado: resuelto.** Los nombres largos podían quedar reducidos a una columna
de un carácter porque el texto completo de la acción ocupaba la segunda
columna de la tarjeta. La rejilla reserva ahora un ancho útil mínimo para el
título y muestra la acción breve «Ver procesos». El nombre completo se
conserva en su etiqueta accesible.

Se verificaron las 68 categorías a 1440 y 1024 px, también con texto aumentado,
sin títulos verticales ni pérdida del nombre accesible.

### H-02 — Recibo del asistente de llamamientos

**Estado: resuelto.** Tras confirmar el paso final, el recibo recibe el foco y
se desplaza al centro del área visible. Si el navegador solicita reducción de
movimiento, el desplazamiento es inmediato; en el resto de casos es suave. La
región conserva su anuncio accesible y puede operarse por teclado.

### H-03 — Estado y acciones de tablas operativas

**Estado: resuelto.** La causa no era solo el contenedor general: varias tablas
densas no reservaban espacio gobernado para Estado y Acciones y algunos chips
de estado impedían el ajuste del texto. Las tablas prioritarias ahora:

- mantienen Estado y Acciones visibles mediante columnas fijas;
- permiten desplazar horizontalmente las columnas de contexto;
- envuelven estados largos dentro de su propia celda;
- preservan controles completos y de tamaño operable;
- exponen el contenedor desplazable como una región accesible y enfocable.

El auditor automático mide celdas, controles y contenido de los chips, detecta
solapes y comprueba cada fila, no solo la cabecera. La corrección cubre
Convocatorias, Solicitudes, Alegaciones, Importación, Dietas y el registro de
versiones de Reglas.

En la vista Reglas, el resumen secundario se apila bajo la tabla a 1024 px. De
este modo la zona principal pasa de unos 436 px a unos 750 px, mientras que la
composición de 1440 px se mantiene sin cambios.

### H-04 — Huellas sintéticas de borradores

**Estado: resuelto para presentación.** Se sustituyeron las cadenas de relleno
por valores hexadecimales SHA-256 fijos y variados. Siguen siendo datos
sintéticos y no acreditan el contenido de un expediente.

**Deuda expresamente documentada:** en el adaptador efímero actual la huella de
estado no se recalcula al editar una revisión. Si la demostración de borradores
evoluciona hacia persistencia real, el adaptador definitivo deberá obtener la
huella del contenido canónico almacenado, no de una constante de presentación.

### H-05 — Reglas y baremación compartían contenido

**Estado: resuelto.** «Motor de reglas configurable» dispone de una superficie
propia para gobierno, catálogo, historial de versiones, aprobación y creación
de una nueva versión. «Baremación y ranking» conserva únicamente el contexto
de cálculo y resultados. Una prueba de regresión exige que ambas vistas tengan
contenido y acciones diferentes.

### M-01 — Paleta parcialmente centralizada

**Estado: resuelto en el alcance detectado.** Los acentos naranja y cian de los
indicadores KPI se obtienen ahora de variables del tema principal. El cambio
mantiene contraste WCAG AA y elimina esos colores aislados del componente.

La centralización completa del diseño continúa siendo una regla del portal:
los módulos heredan tokens y componentes del tema; cualquier excepción debe
declararse y justificarse.

## Correcciones documentales añadidas durante la revisión

La acción de certificados del área aspirante ya no simula un fichero. Genera
en memoria un PDF binario real de demostración que contiene:

- imagen vectorial institucional de la Diputación de Granada;
- título y texto de certificación correspondientes al certificado elegido;
- datos sintéticos de la actuación y referencia opaca individual;
- marca inequívoca «DOCUMENTO DEMO · SIN EFECTOS ADMINISTRATIVOS»;
- QR vectorial con una URL del mismo origen para comprobación.

El QR solo incorpora la referencia opaca `DEMO-REC-NNNN`; no incluye nombre,
identificador personal, expediente ni otros datos privados. El generador
rechaza referencias semánticas, destinos ajenos al origen, parámetros no
permitidos y certificados fuera del catálogo visible. En la demostración solo
se ofrece PDF porque es el único formato que se genera realmente. Los demás
formatos pertenecen al futuro conector documental y no se aparentan en esta
superficie.

## Evidencias y puerta de cierre

Las comprobaciones de código y geometría se conservan en pruebas Node, Python y
Go. La evidencia visual completa se genera fuera de Git en
`var/revision-web/`; su índice legible es `var/revision-web/informe.md`.

Antes de desplegar esta corrección se exige:

1. pruebas Node completas sin fallos;
2. `python3 -m unittest scripts.tests.test_capturar_presentacion_web`;
3. `go test ./...`;
4. verificación del árbol web productivo y `git diff --check`;
5. reconstrucción limpia de la composición Docker;
6. recorrido Playwright completo sin hallazgos;
7. inspección humana de las capturas de 1440 y 1024 px afectadas.

La puerta final obtuvo **278/278 pruebas web**, **22/22 pruebas del capturador**,
la suite Go completa, `go vet`, compilación y verificadores de artefactos sin
fallos. Tras reconstruir Docker, Playwright recorrió **183/183 escenarios** y
generó 183 capturas sin hallazgos. La inspección humana confirmó las pantallas
afectadas en 1440 y 1024 px. El auditor restaura ahora el desplazamiento interno
del menú después de comprobar todos sus controles y la marca institucional queda
fija al desplazarlo. El QR se renderizó desde el PDF final y un lector
independiente recuperó exactamente la URL canónica de comprobación.

El detalle reproducible se registra en `revision_web_presentacion.md`.

## Cierre para dirección

La antigua propuesta T22 queda **cerrada dentro de la entrega de
presentación**: H-01 a H-05 y M-01 están corregidos y protegidos por pruebas.
No se modifica el dominio, los puertos hexagonales ni los adaptadores
productivos; los cambios pertenecen al adaptador web y al instrumental de QA.
La demostración sigue sin equivaler a una puesta en producción ni autoriza el
tratamiento de datos reales.
