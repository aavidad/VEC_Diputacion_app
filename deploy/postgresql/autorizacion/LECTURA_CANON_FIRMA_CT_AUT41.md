# Lectura histórica del canon de firma CT

AUT41 añade una función privada que devuelve los bytes de la evidencia
nominal conservada por AUT32. Exige el comprobador de consumo de lectura de
AD178, que desde AD193 acredita el consumo y la auditoría de la misma
transacción por el sello `transaccion_origen`, no por `xmin`. Se aplica con
`deploy/principal/lista_sql_claude_firmas_recuperacion_20261005.txt`, después
de AD178 y AD177.

`recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)` recibe un selector
histórico de nueve claves, los bytes exactos de la consulta de firmas V2 y el
consumo actual de esa lectura. Solo el propietario CT puede ejecutarla. El
selector contiene `evidencia_ref`, `canon_sha256`, `efecto_original_ref`,
`organizacion_ref`, `unidad_ref`, `expediente_ref`, `documento_ref`,
`consumo_original_decision_ref` y `consumo_original_huella_sha256`. CT debe
resolverlos de su fila de firma V2, sin aceptarlos del navegador.

La función llama primero a
`vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2`.
El comprobador debe acreditar la autorización de lectura vigente, sus 48
campos y el consumo y la auditoría de la misma transacción. AUT41 coteja
organización y expediente con el selector, bloquea con `FOR SHARE` la fila
AUT32 exacta y compara sus referencias, huella y consumo originales. La
referencia de evidencia se vuelve a comprobar con la fórmula AUT32:
`evidencia:competencia-firmante-ct:` seguida de SHA-256 de los bytes UTF-8 de
`efecto_original_ref` concatenados con el canon original. Se devuelve el
`bytea` guardado, sin reserializarlo.

`documento` de la consulta actual es la clave del tipo documental, por ejemplo
`informe_definitivo`. `documento_ref` de AUT32 identifica el PDF original.
AUT41 no los equipara: CT debe acreditar en su propia historia la relación
entre clave, versión de expediente y PDF. La unidad procede del selector CT y
se coteja con AUT32; el material de consulta V2 no la incorpora. AUT41 tampoco
exige que el cargo, la asignación o el certificado del firmante anterior sigan
vigentes. El actor actual autorizado pertenece al circuito de lectura AD178.

El único llamante previsto es CT175, que forma el selector con sus propias
filas y ya filtra por tipo documental y versión. AUT41 por sí sola sólo liga el
canon a organización y expediente: cualquier otra función del propietario CT que
la llame tiene que hacer los mismos cotejos.

La prueba SQL adjunta comprueba estructura, propietario, ACL y rechazo de un
selector vacío. No fabrica un consumo positivo. Se ensayó el 5 de octubre de
2026 en el clon de la principal posterior a AD193 (repetido sobre main con AD195 y AD196) (con AD178 y AD177): UP
único con código 0 y prueba verde. Falta el recorrido causal con un consumo
real de recuperación, que depende de la fuente nominal y del montaje CT.
