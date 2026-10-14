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

El orden final es consolidada común y fachadas → Dietas13 → Dietas17 → Dietas19 →
catálogos publicados. Documentos21 y su autoridad común van antes de Dietas16
y de la subida de justificantes. Hasta que estas dependencias estén
identificadas, revisadas y ensayadas juntas, el corte no está listo para
instalar.

## Lectura propia de la asignación en Personal

Dirección aprobó esta lectura el 15/10 a las 01:40. Personal conserva la
consulta anterior y su recurso de cuatro ámbitos. Dietas usa el contrato
propio desde su GET y desde la preparación de edición y envío, con el mismo
contexto registrado y perfil de la petición. No suma perfiles.

| Elemento | Valor |
| --- | --- |
| Acción | `personal.asignacion_dietas.propia.consultar` |
| Finalidad | `preparar_borrador_dietas` |
| Audiencia | `vec_personal.asignacion_dietas.propia.consultar.v1` |
| Perfil nominal | `consulta_propia_asignacion_dietas_personal` |
| Módulo y tipo | `personal` / `asignacion_dietas_propia` |
| Referencia | Relación exacta seleccionada, `rel_…` |
| Ámbitos | Solo `persona_ref` y `empleado_ref`, derivados del actor acreditado |
| Atributos | `relacion_ref`, `unidad_ref`, `fecha_referencia`, `operacion`, `material_sha256` |
| Material | `vec.personal.asignacion-dietas-propia.v1`; operación `consultar_propia` |
| Garantía y superficie | `alto` / `interna_corporativa` |
| Obligaciones | Ninguna adicional |
| LOGIN y grupo exclusivos | `vec_personal_d7_asignacion` / `vec_personal_d7_ejecutor` |

La fachada común nueva es
`vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_propia_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)`.
Concede ejecución solo al propietario de Personal. Devuelve el mismo contrato
de constancia que la consulta anterior: decisión, efecto, huella del efecto,
huella de consumo, auditoría, instante y marca de consumo nuevo.

Los 17 campos exactos autorizados, ordenados, son:

```json
["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]
```

Personal53 añade `vec_personal.consultar_asignacion_dietas_propia_v1`, con la
misma firma de once argumentos y respuesta de 17 columnas que la función
existente. Comprueba titular, relación activa y unidad; consume y audita la
decisión en su transacción antes de devolver la proyección. Dietas conserva
la revalidación de esa asignación en la transacción posterior que guarda o
envía. Un cambio entre ambas lecturas debe denegar el efecto.

Esta pieza requiere la consolidada común de C antes de Personal53. Su contrato
publicado no acredita todavía instalación ni aceptación en Chrome.
