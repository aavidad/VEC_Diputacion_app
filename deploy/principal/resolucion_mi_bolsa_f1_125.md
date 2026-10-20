# Resolver una aceptación recibida en Mi Bolsa

Tarea F1-125. Código en preparación; instalación, revisiones y recorrido final
pendientes. Dirección confirmó su alcance el 20/10 a las 12:42.

RRHH revisa la aceptación firme de Mi Bolsa y confirma su resolución desde la
ficha. El justificante de Bolsa y el recibo de Contratación conservan sus
referencias y fechas propias. La vía de correo mantiene su formulario.

## Preparar la copia

Reutilizar la fábrica y la etapa de firma de `LEEME_copia_local.md`, apartados
2, 6, 8 y 11, con material y recibos de la propia copia. No activar
`--portal-externo` para Mi Bolsa interna. Crear un expediente de vacante con
necesidad RPT guardada y recorrer los pasos previos del caso 4.0: los
expedientes antiguos de semilla no acreditan esa relación.

La instalación incorpora CT249 mediante la lista SQL ordenada y reinicia la
aplicación. No añade variables de configuración. La aplicación también debe
arrancar sin CT249; en ese caso solo esta resolución queda indisponible.
La cabeza, huellas y comprobaciones de activación se fijarán tras el ensayo.

## Comprobar la resolución

1. RRHH pulsa «Llamar a la siguiente persona» desde la ficha. Deben conservarse
   el recibo de selección de la bolsa constituida y el vínculo de su emisión.
2. Antonio acepta desde Mi Bolsa y conserva el recibo de respuesta firme.
3. RRHH abre esa aceptación en la ficha, comprueba la respuesta y el plazo y
   pulsa «Confirmar aceptación». Debe obtener 201 y el recibo de resolución.
4. Reiniciar aplicación y PostgreSQL; recargar la ficha y comprobar las mismas
   referencias y fechas, sin duplicados. Un reintento conserva la operación.

El servidor deriva la selección original, consume la decisión de autorización
y verifica la aceptación por la función nominal de Bolsa en la transacción
que registra el efecto. No se crean comunicaciones ni justificantes de correo.
La pantalla solo ofrece la acción para el origen nativo acreditado; una emisión
manual no basta. Tampoco elige la primera persona entre varias participaciones.

El plazo queda como revisión expresa de RRHH: esta vía no acredita un cálculo
automático de vigencia. La consulta ordinaria recupera referencia de resolución,
recibo, fecha y respuesta; no expone actor, política ni auditoría completos.
La propuesta de formalización posterior sigue exigiendo correo; su adaptación
corresponde a F1-127, el siguiente corte asignado por dirección.
