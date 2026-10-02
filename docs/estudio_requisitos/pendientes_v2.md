# Pendientes V2 del baremador

El código preparado incluye el cálculo exacto de experiencia y las familias básicas de formación, titulaciones y otros méritos. Usa reglas e instantáneas identificadas por sus huellas. El editor local compara borradores con los mismos servicios Go; Provisión añade una simulación propia de grado, trabajo desarrollado por nivel, antigüedad, permanencia, cursos y titulaciones, con tablas y topes editables. Son simulaciones con datos sintéticos, no una capacidad institucional instalada. La valoración oficial aún requiere lectura atestada de la versión gobernada, autorización, confirmación durable y auditoría; esa cadena es una dependencia del producto, no una mejora opcional de esta lista.

Las ampliaciones siguientes se harán por convocatoria y contra sus bases aprobadas:

- Ampliar las familias básicas con incompatibilidades, equivalencias y tratamiento de evidencias previstos en las bases. La familia V1 excluye el título marcado como requisito y bloquea duplicidades; no cubre bases que permitan otro tratamiento.
- Incorporar las políticas de jornada por horas, solapes favorables, mínimos de sección y tope global que el cálculo de experiencia V1 rechaza expresamente.
- Conectar el editor local al portal interno con su autoridad real, casos de conformidad aprobados y el circuito de revisión y publicación. Reutilizar el puerto de gobierno y su control de versión; la activación sigue deshabilitada.
- Completar la valoración administrativa y la adjudicación global de Provisión con preferencias y desempates publicados. Su simulador conserva reglas propias; no reutiliza resultados de Bolsa.
- Ampliar la conversión por tramos a meses civiles o años sólo cuando las bases definan cómo acumular periodos y restos. Provisión V1 la rechaza; permite esos métodos por periodo con descarte explícito. No convertir meses en treinta días ni unir servicios implícitamente.
- Reunir experiencia y los demás méritos en una valoración completa por convocatoria. Los modos actuales conservan contratos separados y muestran sus desgloses y bloqueos, sin puntuar definitivamente.

Los coeficientes y fechas de los ejemplos sintéticos no aprueban bases institucionales. No se importan datos del prototipo local. Cada nueva familia necesita ejemplos aprobados para su convocatoria.

## Contraste que guía el corte

La [Bolsa Única de Andalucía](https://www.juntadeandalucia.es/boja/2019/240/18.html) fija coeficientes y topes en su convocatoria. La [Ventanilla Electrónica del SAS](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales/como-cumplimento-una-solicitud-de-autobaremo) separa cálculo, revisión y presentación. [SAP SuccessFactors](https://help.sap.com/docs/successfactors-performance-and-goals/implementing-and-managing-performance-management/rating-calculation) y [Odoo](https://www.odoo.com/documentation/17.0/applications/hr/appraisals.html) muestran criterios y escalas configurables por proceso. Se adopta la separación entre regla, cálculo y decisión; ninguno de esos productos aporta una fórmula aplicable por defecto a VEC.

## Original de la primera firma local de Contratación

El verificador acredita que la firma cubre el original aportado por el cliente.
Falta cotejarlo en backend con el borrador exacto de VEC por expediente, versión,
tipo documental y versión de plantilla antes de verificar y custodiar. Resolver
o conservar ese original por su autoridad; regenerar desde el catálogo vigente
no basta. La firma local mantiene `firma_eficaz=false` y no sustituye al circuito
corporativo. Los certificados distintos de una misma persona necesitan una
relación nominal aprobada; el canal de desarrollo exige el mismo certificado
para autenticarse y firmar.

## Cofirma PAdES de un mismo PDF

El diseño aprobado conserva el visto bueno de Dirección o Jefatura y la firma
de la Diputada como dos PDF distintos sobre un mismo original, que debe quedar
custodiado antes de firmar. Cada PDF tendrá un firmante verificado, su recibo
y su referencia de custodia. No se presentará como un PDF con dos firmas.

Si más adelante se necesita una cofirma incremental en un único PDF, GrxFirma
y el puerto de verificación deberán identificar el certificado de cada firma
con su revisión y `ByteRange`, comprobar que las firmas anteriores siguen
válidas y rechazar cambios no permitidos. El dictamen actual solo acredita un
firmante y rechaza la identidad ambigua en un PDF con varios. Este cambio queda
para V2; no condiciona el recorrido de dos evidencias separadas aprobado por
dirección el 2 de octubre de 2026.
