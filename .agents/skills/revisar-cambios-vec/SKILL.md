---
name: revisar-cambios-vec
description: Revisa cambios VEC de forma independiente y acotada. Usar para SQL, identidad, seguridad, consistencia, revisiones de integración o segunda revisión sensible.
---

# Revisar cambios VEC

Trabajar sobre una candidata identificada por commit o hashes. Leer el diff y las
piezas que determinan su comportamiento. Conservar el modo de sólo lectura.

1. Identificar contrato y efecto prometido. Revisar el recorrido de datos y autoridad.
2. Buscar defectos concretos: duplicados, escritura parcial, pérdida de historia,
   permiso incorrecto, respuesta engañosa, referencias sustituidas o error de montaje.
3. En SQL comprobar transacción, aislamiento, permisos, consumidores y migraciones ya
   instaladas. En identidad comprobar actor, ámbito, vigencia y consumo de autoridad.
4. Distinguir hallazgo reproducible de mejora opcional. Proponer corrección acotada.
5. Emitir GO o NO-GO para la candidata exacta con ubicación, gravedad y efecto observable.
   Un GO de código no autoriza instalación ni acredita un recorrido no ejecutado.

En zona sensible aportar una de las dos revisiones independientes. El segundo revisor
recibe artefactos y requisitos, no debe limitarse a confirmar la conclusión del primero.
Si delegas, transmitir sólo lectura y mantener independencia respecto al productor.
No ejecutar migraciones ni pruebas con efectos sobre el runtime durante la revisión.
