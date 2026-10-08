# Fuente nominal de cargos de Personal 000028

Personal conserva el catálogo privado y versionado de cargos y sus enlaces con personas. Cada cargo enlaza una revisión vigente del órgano de Organización y, cuando procede, una revisión del puesto RPT. Cada publicación exige un acto, una fuente, sus versiones y sus huellas. Una referencia de perfil de acceso no crea un cargo ni acredita a quien lo ejerce. Las clases titular, delegación de competencias, delegación de firma y suplencia quedan separadas.

El administrador de la aplicación publica con la acción `personal.cargo_competencial.publicar`. La función exige una concesión V3 vigente, versión esperada y huella de la versión anterior. Conserva el acto, un recibo y la referencia a la auditoría común en la misma transacción. Un intento con la misma clave y el mismo material recupera el recibo original tras un nuevo consumo autorizado; una clave reutilizada con otro material se rechaza.

Para firmar en Contratación, AUT32 invoca `vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)` dentro de la transacción `SERIALIZABLE READ WRITE` de CT. Personal compara las referencias, versiones, huellas y vigencias del canon histórico con el cargo y la titularidad publicados. También contrasta el acto de delegación o suplencia y el alcance de acción, recurso y finalidad. Los bloqueos de las versiones y los punteros duran hasta `COMMIT`; las historias P10 y B2 se bloquean para impedir revisiones nuevas durante esa transacción. `enlace_personal` corresponde al titular. Si el acto del delegado exige o declara una ocupación laboral, se comprueba por separado y se devuelve en `enlace_ejerciente_personal`. Un cargo electo puede carecer de esos enlaces sin inventar una relación de servicio.

La fuente registra el dato declarado en el acto. Ni su publicación ni la lectura técnica prueban por sí solas la validez jurídica del acto. La función no publica una ruta HTTP ni asigna automáticamente perfiles. La instalación y el ensayo causal dependen de AUT33, AD165 y AD166; una migración presente en Git no está instalada en la base.

## Corrección de Personal37

Dos CHECK de `enlace_cargo_competencial_historia` (`recurso_ref` y
`finalidad_ref`) usaban la repetición `{2,511}`. PostgreSQL solo admite
repeticiones hasta 255 y compila la expresión al evaluarla, así que todo INSERT
de un enlace fallaba con «invalid regular expression: invalid repetition
count(s)». Se vio al publicar el primer enlace desde vec-admin. Personal37
cambia las dos reglas por la misma intención (primer carácter, alfabeto y
longitud de 3 a 512 medida aparte) sin tocar la fachada. Su vector
`pruebas_sql/personal37_restricciones_enlace.sql` (7/7) falla antes de
instalarla y pasa después. Repetirla se para en la preimagen.

## Personal38: enlace por tipo de recurso (4c-4)

Decisión de dirección del 05/10: el enlace de ejercicio de un cargo guarda como
recurso el tipo de recurso del paso del plan (por ejemplo
`documento_contratacion_temporal`), no un documento. Personal38 añade
`localizar_enlace_cargo_ct_v1`, que devuelve el único enlace vigente de una
persona para un cargo, acción, tipo y finalidad (solo lo ejecuta
`vec_autorizacion_propietario`, y la sesión debe ser del ejecutor CT). También
cambia `leer_revalidar_cargo_ocupante_ct_v1` para que, si el contexto trae
`recurso.tipo_recurso`, compare con él el recurso del enlace. La decisión y el
consumo de la firma siguen ligados al documento exacto en CT172/AD170. Vector
`pruebas_sql/personal38_localizador_enlace.sql`: 7/7 tras instalar; antes de
instalar falla.

Estado entre el corte 1 (Personal38) y el corte 2 (AUT, sucesor de AUT35): la
firma nominal de Contratación temporal queda cerrada para cualquier forma de
enlace. AUT35 deniega un enlace por tipo (sigue cotejando con el documento) y
AUT32, a través de Personal28, deniega un enlace por documento. Antes de
Personal37 no se podía insertar ningún enlace, así que no se rompe nada que
funcione; conviene instalar los dos cortes seguidos.
