# Actos de firma y plan nominal de la copia

`preparar_actos(c)` toma la semilla privada de firmantes de esta copia y la identidad de las dos personas administradoras del arranque H10. Selecciona para cada una su perfil de Aplicación. Prepara cinco cargos, cinco enlaces de titularidad y cuatro vínculos de certificado; coteja el certificado y conserva los bytes de cada petición. ADMIN 1 publica cada acto mediante su ruta nominal. Cada respuesta confirmada deja un recibo privado en `recorrido/actos/recibos/` y el conjunto queda en `recorrido/actos/metadata.json`.

`preparar_plan(c)` exige los 14 recibos, comprueba la correspondencia vigente entre la organización operativa y la canónica, y construye cinco entradas ligadas a esos cargos. Ejecuta el productor `vec-plan-firma-validar --config … --salida …` de esta candidata. ADMIN 1 crea el plan y ADMIN 2 lo publica. Los materiales, huellas y recibos quedan en `recorrido/plan/`. Tras ambos recibos se guarda el paquete en `material/identidad/firma-recursos/plan-firma-publicado.json`.

Una marca `*.sin-confirmar.json` significa que se desconoce el efecto de un POST. Hay que conciliar esa operación por su clave y sus mismos bytes antes de continuar. El recorrido no crea otra clave ni reenvía el acto mientras la marca exista. Una reentrada con recibo confirmado coteja el material original y continúa sin repetir el POST.

La provisión de roles AUT53 de la semilla responde a la excepción de Dirección de las 14:52. No acredita gobierno técnico de roles. Los catorce actos y el plan sí pasan por dos personas ADMIN distintas, con perfil activo de Aplicación. Ninguno de estos archivos acredita una firma de documento, una capacidad por sí sola ni una publicación en cidonia. La instalación, el arranque y el recorrido en Chrome corresponden a la candidata completa de la copia local.

Prueba focal sin llamadas reales: `PYTHONPATH=deploy/principal python3 -m unittest deploy/principal/test_recorrido_copia_actos_plan.py -v`.
