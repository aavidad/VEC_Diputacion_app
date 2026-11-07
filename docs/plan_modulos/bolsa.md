# Bolsa de trabajo: plan vigente

Corte del 20 de octubre de 2026, sobre `origin/main@3ef7dee0b`.
Bolsa y Contratación temporal son la prioridad de la fase 1. Se trabaja en
copias locales con datos ficticios; cidonia está congelada. La cola y sus
responsables están en el [tablero](../producto/TABLERO_TAREAS.md).

## Integrado y comprobado por partes

- El módulo conserva convocatorias, candidaturas, orden, disponibilidad,
  llamamientos y respuestas. Contratación consume sus contratos; no mantiene
  otra bolsa ni decide su orden.
- El [ensayo de dos plazas](../../deploy/principal/aceptacion_dos_plazas_f1_55_20261018.md)
  acredita dos aceptaciones previas, adjudicación por orden y recuperación de
  recibo y fecha tras reiniciar, sin duplicar el acto. No responder no causa
  por sí solo una baja.
- La [entrada externa a Mi Bolsa](../estudio_requisitos/cuenta_externa_f1_46_20261020.md)
  se comprobó antes y después del reinicio. La cuenta no tenía participaciones:
  esa prueba no acredita inscripción, respuesta ni subida de justificantes.
- El [cese anterior al vínculo](../estudio_requisitos/recorrido_f1_30_cese_sin_vinculo_20261020.md)
  tiene ensayo propio. No acredita la vuelta a la bolsa tras el nombramiento
  del expediente del caso único.

## Qué falta para cerrar este recorrido

1. Completar el llamamiento, la aceptación y su resolución en el mismo
   expediente del [caso 4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).
2. Continuar desde la adjudicación de varias plazas hasta los expedientes de
   cada persona, con sus documentos y efectos; no basta el recibo de oferta.
3. Acreditar, después del alta en Personal y el cese, la disponibilidad y el
   orden de retorno en Mi Bolsa y en la consulta de RRHH, también tras reiniciar.
4. Recorrer la renuncia justificada con PDF: aportación, custodia, consulta y
   descarga por RRHH. No atribuir eficacia a una referencia sin fichero.
5. Cumplir la medición de rendimiento y evitar fallos de consulta o comunicación
   en el recorrido. El [acta vigente](../producto/acta_rendimiento_lecturas_f1_34_20261016.md)
   conserva las limitaciones bajo concurrencia.

## Activación y decisiones pendientes

La [receta de copia local](../../deploy/principal/LEEME_copia_local.md) reúne
preparación, permisos, catálogos y reinicio. El correo local solo admite las
direcciones ficticias de la copia; su recibo no prueba entrega corporativa.
Las decisiones internas pendientes de RRHH o Protección de Datos siguen en
[dudas.md](../../dudas.md), con valores provisionales configurables cuando
corresponde. Las preguntas resueltas no se vuelven a plantear.

El [plan general](../producto/PLAN_IMPLEMENTACION.md) fija el cierre de fase;
el [plan de Contratación](contratacion_temporal.md) explica la continuación.
El [inventario anterior](../historico/plan_modulos/bolsa_anterior_20261020.md)
se conserva como antecedente, sin mantener su antigua cola de PR o subidas.
