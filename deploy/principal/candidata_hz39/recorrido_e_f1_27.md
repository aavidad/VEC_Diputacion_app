# F1-27: recorrido en la copia E

E se creó el 21/10/2026 desde `df8d76a0ce7c7c3ef32e90df15387e76dac0d46b`:
`origin/main@4bc37cb1a80a19381f3fe0b3875135ced38654fd` más la hoja inicial de
este ensayo. Incluye la cadena F1-131 y el preparador nominal de #1396.
No incorpora ramas pendientes. La fuente de ejecución permanece congelada;
esta hoja se mantiene en una rama de entrega separada.

Se empleó la orden completa de
[`recorrido_copia/LEEME.md`](../recorrido_copia/LEEME.md), con una instancia
nueva del equipo P y cinco puertos locales propios. El respaldo común se
abrió solo para lectura. La orden terminó con código 0 en 275,3 segundos y
guardó el recibo final de fábrica. Certificados, conexiones, recibos completos
y registros se conservan fuera de Git.

## Composición y activación

- Aplicación, Administración y productores: misma revisión congelada.
- SQL aplicada una vez: 138 migraciones de la lista elegida por la receta.
  Huella de `sql_aplicado.json`:
  `8d864db01d866b299eee9ae3048975aa017efe1cef56942f63d310aef8486921`.
- Binario de la aplicación:
  `191d00e407ffb69d42d6f5e722d89fb1ee4fb750ec6cc2c2eec57417c3bccc47`.
- Verificador GrxFirma:
  `e0b316575010b8bb8891305c91b3ad3926ba9fbaccdaf3f947a79f8ad53dd322`.
- Manejador de firma:
  `89a7ecb8bcddd1501873a5debb4fb777f535ca7a62f0cc73be62c927a0403b36`.
- Tramitación: certificado operativo original de RRHH. Firma del informe:
  cuarto certificado nominal propio. Cada firmante tiene un entorno aislado.
- No se añadieron permisos, selectores ni correcciones manuales a la receta.
- D se borró con el gestor público después de que E superara el paso 3.

## Resultado por paso

Caso de aceptación: `docs/producto/FLUJO_APP.md`, apartado 4.0.
Expediente de E: `2026/95029`, puesto 9, plaza 2026/0033, una persona,
periodo del 21/10 al 31/12/2026. Todos los actos siguientes se hicieron
mediante formularios de Chrome del sistema, con Playwright.

| Paso | Resultado en E | Evidencia |
| --- | --- | --- |
| Creación completa | Confirmada | Código 0; recibo de fábrica |
| 1. Petición del centro | Confirmada | POST 200; pendiente de ratificar, v1 |
| 2. Ratificación | Confirmada | POST 200; ratificada, v2 |
| 3. Alta del expediente | Confirmada | POST 201; solicitud/en_curso, v1 |
| 4. Análisis | Confirmado | Formulario visible con técnica operativa; POST 201, v2 |
| 5. Bolsa vigente | Confirmada | Decisión de cobertura POST 201, v3 |
| 6. Asignación | Confirmada | POST 201, v4 |
| 7. Informe jurídico | Confirmado | POST 201, informe_juridico/en_curso, v5 |
| 7b. Firma del técnico | Bloqueada antes de firmar | Consulta y original PDF 200; programa nativo sin abrir |
| Documentos de la bolsa | Consulta correcta | Técnica operativa: GET 200; acta de constitución visible |
| Archivo documental de la ficha | Bloqueado | POST de consulta 503; incidencia de gobierno |
| Borradores de la ficha | Sin montar | Capacidad declarada sin_montaje; no se pide la lista |
| 8–15 | Pendientes | Requieren completar las firmas del informe |
| Reinicio de aplicación y PostgreSQL | Correcto | Chrome conserva v5 e historial; instantáneas iguales |

Las identidades nuevas del centro se llaman Lucía Navarro Molina y Diego
Serrano Vega, según la receta. No se cambiaron sus nombres para hacerlos
coincidir con los del caso escrito.

## Bloqueos enviados a dirección

1. **Borradores sin activar.** La receta omite
   `VEC_CT_PLANTILLAS_DOCUMENTAL_ENABLED`, aunque activa Documentos y prepara
   el catálogo de plantillas. El detalle declara
   `capacidades_ficha.borradores_publicados.estado=sin_montaje`.
   `contratacion_temporal_plantillas_desarrollo.go:111` exige ese selector;
   `vista-expedientes.js:363` muestra el aviso sin pedir la lista.
2. **Archivo documental del expediente.**
   `POST /api/vec/documentos/expedientes/consultas` devuelve 503,
   `api.documentos.error.servicio_no_disponible`. El registro declara
   `GOBIERNO_V3_NO_DISPONIBLE` antes de ejecutar la lectura de Documentos.
   El rol documental publicado por la receta tiene cinco concesiones y
   ninguna para `documentos.expediente.listar` (cotejado en PostgreSQL).
   El contrato de esa lectura está en `documentos_montaje.go:296`.
   La causa concreta de la emisión fallida queda para el equipo dueño;
   no se confunde esta lectura con la consulta correcta del acta de la bolsa.
3. **Manejador de firma.** El helper `entorno_firma.py` escribe siempre el
   ejecutable entre comillas en `Exec`. El `xdg-mime` del sistema extrae ese
   campo con `first_word`, que declara no interpretar comillas, y lo busca
   con `command -v`. Descarta el lanzador propio y elige el manejador del
   sistema. La salida del GrxFirma propio sigue vacía. Se canceló la espera
   del navegador sin registrar firma ni alterar el paquete preparado.

También se observan un 404 de preferencias de Usuarios para la técnica
operativa y un 401 para el técnico nominal; la receta nueva no activa ese
selector. El certificado operativo recibe 403 al consultar firmas nominales;
el nominal del técnico consulta las mismas firmas con 200. No se amplían
permisos para eliminar esas denegaciones.

Se conservan ocho respuestas confirmadas de los pasos recorridos y del PDF
original, con referencias, versiones y fechas. El informe original quedó
custodiado; no se ha firmado. No se acredita la resolución, ninguna de sus
firmas ni los pasos posteriores de F1-27 o F1-28.

Tras reiniciar con el gestor público, el expediente conserva la versión 5 y
las cinco actuaciones en Chrome. La instantánea de la fila vigente y de ocho
recuentos de firma/fiscalización es idéntica antes y después. La ficha no
desborda horizontalmente a 1440 ni a 390 píxeles. Esta comprobación parcial
no acredita los pasos 8 a 21.

El técnico nominal volvió a pulsar «Comprobar opciones de firma» en Chrome
después del reinicio. La recuperación del original devuelve 200 con los mismos
bytes PDF, referencia, huella y comprobante de custodia que antes. No se
inició otra firma.
