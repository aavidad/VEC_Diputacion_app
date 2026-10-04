# Contexto ADMIN nominal — CA36

CA36 prepara el registro y la recuperación de un contexto V2 para una petición
ADMIN concreta. La cuenta privilegiada, el perfil seleccionado y la sesión
deben seguir vigentes después de las últimas esperas. El vínculo IS procede
de la misma petición y se comprueba por referencia, versión y huella exactas;
buscar la última sesión de una cuenta no acredita esa relación.

La fachada CA tiene un grupo de ejecución exclusivo. El LOGIN CA aún no está
aprovisionado; deberá carecer del grupo general de ContextoActor y de permisos
sobre sus tablas. El cuerpo medido
del núcleo V2 se extrajo a dos helpers privados del propietario CA. Las entradas
generales conservan su acreditación, su propietario, configuración y ACL. Las
fachadas ADMIN comprueban su grupo propio y el vínculo IS16 exacto antes de
llamar a los helpers. Estos conservan la lógica de operación, recibo y CAS.
En ADMIN el alcance de proyecciones será vacío, el método
será certificado o DNIe y la garantía será alta.

El adaptador Go exige `NuevoContextoRegistradoADMINPostgreSQL` con pool CA
propio y proceso privado validado para AD192. El constructor anterior sigue
compilando, pero permanece cerrado. La composición del pool nuevo queda fuera
de este corte.

`registrar_contexto_admin_v1` usa `SERIALIZABLE`. IS16 coteja el método real
y, en ese aislamiento, comprueba el perfil de Aplicación por CA31. CA36
verifica además la selección y las versiones actuales de su propio contexto.
La recuperación de un COMMIT incierto usa `READ COMMITTED` con la operación,
recibo, vínculo y evento originales. Sólo lee el enlace CA y coteja el evento
existente en AD192. Una consulta funcional posterior usa `SERIALIZABLE` y un
evento nuevo; no se confunde con la recuperación.

Una tabla CA inmutable enlaza operación y recibo con referencia, versión y
huella de vínculo IS16, autenticación, sesión y evento AD192. Es dato del
dominio de contexto, no otra cadena de auditoría. El resultado favorable
necesita el evento común AD192 confirmado en la misma transacción que el
contexto. Si el núcleo falla después de empezar el efecto,
la escritura se revierte en una subtransacción y se registra el resultado
negativo antes de confirmar. El adaptador retiene el evento original sólo
durante esa llamada, sin enviarlo a HTTP ni escribirlo en logs. Si la
respuesta auditada fue negativa, la recuperación devuelve ese estado sin
contexto V2; no afirma que no haya otros registros históricos.

## Resultado estructural del 4 de octubre de 2026

El frío posterior a AD192 se restauró en un clon aislado. CA31 e IS14 se
instalaron una vez antes de IS16; IS16 también terminó su UP con salida 0.
Después, CA36 `5f61de133bfac470a34d5cd8716017c8aef276b9`, SQL SHA256
`6dfe1b8a3a6fcdce21a58690049745a0f37b5ee7862a8be4fe016f8beec62c3f`,
terminó su UP con salida 0. `ca36_runtime_acl.sql` terminó con salida 0.
Dos revisiones independientes Sol/Astra cubrieron el candidato; la instalación
estructural se limita al clon y no acredita un contexto favorable.

CA36 reemplazó las dos entradas generales del núcleo CA por fachadas con la
guarda de runtime prevista. Sus cuerpos de negocio pasaron a los helpers
propietarios. El cotejo anterior y posterior confirmó que ambas entradas
conservan propietario, ACL y configuración, y que
`exigir_runtime_contexto_actor_v1()` mantiene además su cuerpo y definición.
La cadena de auditoría conserva sus 6.254 filas y cabeza. El núcleo AD192
conserva su huella de fuente
`b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45`;
el CHECK `auditoria_tipo_disjunto_v4` conserva
`0f6d15ebdc61ba5ff67903bde824db878a6396fa593029e98498946e2d8d1331`.
No se reaplicó AD192 ni se ejecutó DOWN.

Las pruebas Go usan transporte sintético: comprueban la ligadura del acuse y
la recuperación de un COMMIT incierto. Falta un nuevo juego sintético por el
circuito oficial de fuentes y bootstrap, con originales privados en modo 0600;
las asignaciones anteriores han caducado y las identidades originales no se
reconstruyen. No se acreditaron vínculo, sesión o contexto favorables, evento
nominal, LOGIN y pools segregados, garantía alta, PDP, montaje ADMIN ni
recorrido HTTP.

El ensayo pendiente debe comprobar con datos sintéticos: registro exacto y
repetición con el mismo evento sin otra fila ni auditoría; sesión diferente de
la misma cuenta rechazada; retirada del perfil o vencimiento mientras espera
un bloqueo; denegación auditada sin contexto; recuperación tras COMMIT incierto
del resultado positivo y del negativo con el evento original; ausencia de
acuse sin éxito; y acceso directo al núcleo o a la tabla denegado al LOGIN CA.
