# Relación de Personal para RPT: borrador

AD154 y Personal27 preparan una lectura nominal de una relación laboral para
RPT-005/006. Ambos archivos llevan un paro explícito y la extensión `.borrador`.
No hay migración instalable, DOWN, instalación ni capacidad montada acreditada.

El contrato procede del lector Go `1871cab4e14eabe5394bba072b7a714dd9f25c86`,
con los ajustes de `9f0ae4d688a830e9e1be36291957c45205e112fb`, y del apartado
RPT-005/006 de [Personal](../../../../docs/plan_modulos/personal.md). Personal
aporta el hecho laboral; Organización/RPT decide ocupación, reserva y vacantes.

La consulta conserva el actor, la cuenta, el perfil y el contexto originales.
El empleado objetivo puede ser otra persona si existe una concesión exacta.
La frontera inicial exige la superficie acreditada `interna_corporativa`;
un contexto externo se deniega sin cambiar su perfil.

| Elemento | Contrato nominal |
| --- | --- |
| Consumidor AD154 | `consumir_relacion_para_rpt_v3_atestada`, diez argumentos V3 |
| Perfil técnico del núcleo | `relacion_para_rpt` |
| Acción / audiencia | `personal.relacion_rpt.consultar` / `vec_personal.relacion_rpt.v1` |
| Finalidad / tipo | `conciliar_relacion_laboral_para_rpt` / `relacion_para_rpt` |
| Recurso y efecto | La misma referencia `rel_…` |
| Ámbitos exactos | `empleado_ref`, `organismo_ref`, `relacion_ref` |
| Atributos exactos | `conocido_en`, `material_sha256`, `operacion`, `version_esperada`, `vigente_en` |
| Campos, en orden | `cobertura`, `corte`, `estado`, `periodo`, `procedencia`, `version` |
| Obligaciones | Lista vacía |

Personal27 reconstruye los 17 campos del material
`vec.personal.relacion-rpt.consulta.v1`, operación `relacion_para_rpt`, y la
huella del contexto del recurso. `version_esperada` es un número en ese
material y una cadena decimal en los atributos. El actor se revalida con el
reloj actual, aunque el corte de conocimiento sea histórico.

La fuente es `vec_personal.relacion_servicio_historia`, de Personal17.
Se selecciona primero la última revisión conocida de la relación y después
se cotejan empleado, organismo y versión; no se rescata una revisión anterior
para obtener coincidencia. El estado y el periodo proceden de la fila real.
`hasta` es una cadena vacía para un periodo abierto. Los indicadores de firma
y eficacia de esta fuente permanecen falsos: la respuesta conserva
`certeza: no_acreditado` y `cobertura: no_acreditada`. Devuelve únicamente
`relacion`, `corte`, `cobertura` y la evidencia nominal de seis campos.

El LOGIN de lectura tendrá una sola membresía directa en
`vec_personal_ejecutor`, con `INHERIT TRUE`, `SET FALSE` y `ADMIN FALSE`.
La fachada AD solo concede ejecución a `vec_personal_propietario`.
Consumo V3, lectura, recibo y auditoría se confirman en la misma transacción
SERIALIZABLE. El consumo precede al bloqueo de la relación, como en Personal19.
El control de generaciones de Personal27 detecta una instantánea obsoleta
frente a nuevas filas en la historia de solo adición.

Los intentos fallidos se registran después del rollback mediante otro LOGIN
y otro pool. Su único grupo, `vec_personal_registrador_intento_relacion_rpt`,
ejecuta `registrar_denegacion_relacion_para_rpt_v1(text,text,text,text)` y no
puede leer la fuente ni la tabla de intentos. Conserva una correlación real,
un motivo nominal (`entrada_invalida`, `denegado` o `no_disponible`), actor
nulo si no está acreditado y relación nula si no es válida. El instante lo
fija SQL. No registra cortes, hechos laborales, errores internos ni una
concesión inventada. Cada lectura requiere preflight; ausencia o fallo
mantienen cerrado el cliente.

AD154 conserva como contrato la preimagen POST-AD149 congelada de ensayo:
definición `8efb8ae6ceffc5d3c3736a1b8b83543dd6a083eb0ae03eb8f0f4a1e2d3e9b232`,
fuente `da14ce5ef628586ebc1280cd62e6b7da1f57fea9c9b6f6f4233fbecd84c558b9` y
CHECK de audiencias `3ed762b21ac10a8b2c8076c4832b57545f1e3ef93ebd41cf11939466e6b36802`.
El modelo es AD149 `9a58bcb9eea7556b96603dd247568d2a1b340af0`.
Esas huellas no acreditan el estado actual de main ni de la principal.
La conversión queda condicionada a verificar la integración y publicación
de AD149 y a fijar la huella exacta del núcleo que Dirección recapture.
La modificación comprueba preimagen, marcas únicas, propietario, metadatos,
ACL y dependencias; su inversa textual debe recuperar exactamente el núcleo
original, incluidos CRN11, Méritos y Baremo.
Las dependencias de Personal se inspeccionan por `pg_catalog`; AD154 no
concede `USAGE` ni lectura de Personal al propietario de Autorización.

Antes de convertir estos borradores en candidatas, Dirección debe recapturar
el núcleo y las audiencias del main publicado con AD144, AD149 y las
extensiones integradas de A/E conservadas, fijar el orden causal AD154 →
Personal27 y obtener dos GO sensibles sobre el contenido final. Quedan el
ensayo PostgreSQL18 en el clon, la provisión privada de ambos LOGIN/pools,
la instalación autorizada, el montaje y la admisión del consumidor real.
El reproductor está incompleto y mantiene PARO. Sus casos preparados no son
resultados de pruebas. Quedan por validar la conservación de la fuente17 y la
revocación actual, y el lector Go con ambos pools reales ante ausencia o fallo
del registro. Dirección también debe fijar la cota de disco del entorno de
prueba. No se han ejecutado pruebas SQL ni recorrido con estos borradores.

Este contrato no acredita grado, antigüedad calculada, vacante, certificado,
firma ni incorporación eficaz. Los demás módulos conservan su autoridad.
