# Resolver una aceptación de Mi Bolsa desde la ficha

Segundo corte de F1-125. Reutiliza CT201, el verificador nominal de Bolsa y
el acto de revisión de respuesta de RRHH; no añade configuración ni permisos.

## Activación y recorrido

Aplicar CT249 una sola vez después de CT250 mediante la lista SQL principal
y reiniciar con la configuración existente. En el caso 4.0, completar la
emisión nativa y el contacto efectivo; Antonio acepta en Mi Bolsa y conserva
su justificante. RRHH abre la ficha, revisa respuesta y plazo y confirma.
Una emisión manual o la marca de origen por sí sola no habilitan el acto.

El POST devuelve 201 con el recibo CT, distinto del justificante de Mi Bolsa.
Una petición incierta se reintenta con la misma clave; el replay devuelve 200
con las mismas referencias, también después de reiniciar app y PostgreSQL.
La recuperación al reabrir la ficha corresponde al tercer corte de F1-125.

El acto consume su autorización en la transacción del asiento y el recibo.
Bolsa acredita emisión, participación y respuesta originales; una restricción
impide resolver dos veces entre correo y portal. El plazo refleja la revisión
de RRHH, no un cálculo automático. No se fabrica correo. F1-127 consume la
pareja de resolución y recibo. Esta guía no acredita el ensayo: deben guardarse
los recibos y comprobar su identidad antes y después del reinicio.

La resolución conserva las dos variantes del ofrecimiento: con retención
y sin ella. Comprueba la confirmación, los cánones y las huellas originales
y su vínculo con la versión fiscalizada. La caducidad posterior de una
retención no invalida la aceptación histórica; no se repite su confirmación.
