# Autoridades comunes que necesita Dietas

Dirección decidió el 14/10 a las 22:25 que C prepara una única migración
consolidada desde la función instalada en HZ37. La PR de Dietas no incluye
otra reconstrucción del núcleo. AD271 se retiró; su versión anterior está
disponible en la historia de `e1047313c` para contrastar los contratos.

| Acción | Finalidad | Quién y para qué recurso |
| --- | --- | --- |
| `dietas.liquidacion_economica.registrar` | `registrar_liquidacion_economica_dietas` | Perfil fijo de liquidación de la unidad, distinto del titular; comisión exacta, tipo `liquidacion_economica`. |
| `dietas.liquidacion_economica.consultar` | `consultar_liquidacion_economica_dietas` | Titular de su comisión o perfil de liquidación/fiscalización de su unidad; comisión exacta, mismo tipo. |

El registro consume audiencia
`vec_dietas.liquidacion_economica.registrar.v1` y perfil nominal
`registro_economico_dietas`. Los campos son la lista cerrada
`["constancia","economia","esquema","snapshot_sha256"]`; no hay obligaciones
adicionales de registro. La consulta admite solo esa lista y auditoría.
Las concesiones no se suman y las políticas restrictivas siguen cerrando
el acceso cuando su efecto no se puede verificar.

## Fachadas requeridas antes de Dietas13

Dietas13 llama a estas funciones del dueño común, con permiso exclusivo para
`vec_dietas_propietario`:

- `vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)`.
- `vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(bytea,bytea,jsonb)`.

La segunda usa el cotejo central
`vec_autorizacion.cotejar_lectura_liquidacion_dietas_actual_v1(jsonb,text,text,text,bytea)`
y la familia auditora `lectura_liquidacion_dietas`. Deben conservar sesión
ligada al certificado, contexto, asignación y publicación vigentes, ámbito
exacto, auditoría e integridad del recibo. Una consulta no sustituye la
autorización de registro al repetir un acto confirmado.

El orden final es consolidada común y fachadas → Dietas13 → Dietas17 →
catálogos publicados. Documentos20 y su autoridad común van antes de Dietas16
y de la subida de justificantes. Hasta que estas dependencias estén
identificadas, revisadas y ensayadas juntas, el corte no está listo para
instalar.
