# Fuente nominal del cargo para la firma CT

Personal29 añade `vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)`. AUT35 la llama dentro de la transacción serializable de CT, después del consumo V3 común. Los argumentos son, en orden, cargo del plan gobernado, enlace de ejercicio del plan, persona acreditada por CA25, organización y unidad acreditadas por CT174. AUT35 debe comparar también acción, recurso autorizable y finalidad con el plan y con su decisión central; la respuesta de Personal no concede permiso.

El enlace se busca por su referencia exacta y versión actual. Si representa delegación o suplencia, la propia fila identifica al titular mediante referencia, versión y huella; Personal29 comprueba esos datos contra el puntero actual. Después llama a la revalidación completa de Personal28, que verifica cargo, órgano, puesto, vigencia, relación y ocupación cuando corresponden. Los bloqueos de punteros y versiones duran hasta el `COMMIT` de CT. Un cambio, ausencia o ambigüedad deniega la operación.

La respuesta conserva el esquema `vec.personal.cargo-ocupante.ct.v1` y todos los campos de Personal28. Añade `accion_ref`, `recurso_autorizable_ref` y `finalidad_ref` para el cotejo AUT, además de `cargo_procedencia`, `enlace_ejerciente` y `ejerciente_procedencia` con acto, fuente, versiones, huellas y recibos de publicación. En caso de titular, `enlace_ejerciente` coincide con `enlace_ocupante`.

Orden causal: instalar Personal28 y su permiso privado antes de Personal29. La migración 29 no modifica 28 ni datos existentes. Su función pertenece a `vec_personal_propietario`; solo `vec_autorizacion_propietario` recibe `EXECUTE`. No hay ruta de consulta directa de CT, LOGIN ni `PUBLIC`.

La prueba SQL focal verifica propietario, ACL y denegación de un contexto vacío. Quedan pendientes el ensayo en el clon por Dirección, dos revisiones independientes del hash final y una prueba positiva con publicación sintética de cargo y enlace, AUT35 y CT. Ningún archivo preparado acredita instalación, firma o recorrido de RRHH.
