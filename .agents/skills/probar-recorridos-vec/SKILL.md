---
name: probar-recorridos-vec
description: Valida incrementos de VEC con pruebas proporcionales y recorridos reales. Usar en Go/JavaScript, navegador, persistencia, recuperación tras reinicio o regresiones concretas.
---

# Probar recorridos VEC

Elegir la evidencia mínima que pruebe el comportamiento solicitado. Un test unitario
verde no demuestra instalación, persistencia ni montaje en navegador.

1. Leer el cambio y su aceptación. Reutilizar pruebas y utilidades existentes.
2. Ejecutar la prueba focal del camino cambiado y del fallo concreto que se corrige.
   No crear baterías que repitan la implementación ni campañas preventivas generales.
3. Agrupar go test ./... y go vet ./... al cierre de un corte; evitar repeticiones por
   archivo. Coordinar con el director una o dos ejecuciones pesadas simultáneas.
4. Para UI conectada, observar petición, estado y recibo reales. Comprobar 390 px y
   escritorio cuando el cambio tenga consecuencias visuales.
5. Para escritura, comparar recibo, fecha, versión e historia tras recuperar. Si el
   corte exige reinicio, usar los servicios autorizados y comprobar que no hay duplicados.
6. Si la operación ya existe, probar su GET o recuperación prevista. No lanzar otro
   POST para superar un error de recuperación ni reiniciar escenarios conservados.
7. Informar comando exacto, resultado, alcance probado y limitaciones. Distinguir pruebas
   pasadas, omitidas por filtro, no ejecutadas y fallidas.

SQL/identidad/cripto/datos personales requieren las revisiones independientes vigentes.
No confundir preparación local con validación final del runtime remoto. Delegar con
$orquestar-vec si hay comprobaciones independientes que aporten evidencia distinta.
