# Antecedentes técnicos del desatasco, 3–19 de septiembre de 2026

Nota histórica conservada el 20/10/2026. Las reglas actuales están en
[AGENTS.md](../../AGENTS.md) y el [plan](../producto/PLAN_IMPLEMENTACION.md).
Las antiguas colas de sesiones, responsables y subidas se han retirado.

El problema de aquel corte era la falta de composición: existían rutas y
casos de uso probados que el proceso de aplicación no montaba. La aceptación
pasó a exigir un recorrido desde el navegador, con autorización, persistencia
y recibo, recuperable tras reiniciar.

La integración con Bolsa exige conservar el orden y el recibo al repetir
una operación. La importación de datos de Convoca no convierte sus datos
previos en una nueva actuación administrativa de VEC.

En las consultas paginadas, el cursor debe conservar el contexto autenticado
para que una página posterior no cambie de identidad. La consulta y la
denegación conservan auditoría; un fallo no se interpreta como lista vacía.
La confirmación local de GINPIX o correo no acredita una entrega externa.

Estos antecedentes explican las condiciones de aceptación actuales; no
prescriben ramas, servicios ni procedimientos de instalación de septiembre.
