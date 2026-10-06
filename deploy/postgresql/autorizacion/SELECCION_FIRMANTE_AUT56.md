# AUT56: selección central del firmante (4c-4, corte 2)

El enlace de un cargo (Personal) guarda el tipo de recurso del paso del plan
(por ejemplo `firma_vec_documento_contratacion_temporal`), no un documento
concreto. Decisión de dirección del 05/10.

- **AUT35** (`construir_contexto_nominal_firmante_ct_v1`) coteja el recurso del
  enlace con `rec->>'tipo_recurso'`. Es una sola línea, sustituida con marca
  única sobre la definición instalada. Propietario, ACL y configuración no
  cambian. La decisión y el consumo de la firma siguen ligados al documento
  exacto: CT172 fija `recurso_autorizable_ref = documento_ref = OriginalRef`,
  y la huella del material y del descriptor entra en la de contexto de la
  decisión V3.
- **`seleccionar_firmante_plan_ct_v1`** la ejecuta solo el ejecutor CT, antes
  del PDP. Lee:
  - la persona del certificado, con CA25 (`leer_revalidar_certificado_firmante_ct_v2`);
  - el enlace vigente del cargo por tipo, con Personal38;
  - la única asignación activa y vigente de esa persona con el rol del paso,
    dentro de la organización y, si la asignación la tiene, de la unidad.

  Devuelve la selección (perfil activo, rol, cargo y enlace), la cuenta y el
vínculo del certificado, y las versiones y
  huellas de la asignación, el rol y el control. No escribe ni concede nada.
  Ninguna o más de una asignación se deniegan.

El vector `pruebas_sql/aut56_seleccion_firmante.sql` da 12/12. Comprueba:
- selección positiva;
- denegación con otro rol, otra unidad, otro certificado y enlace por documento;
- dos asignaciones ambiguas;
- LOGIN ajeno sin EXECUTE;
- AUT35 por tipo;
- con el mismo cargo, una decisión para el original A presentada con el
  material del original B: CT172 la rechaza antes del consumidor V3, mientras
  que con A sí llega a él.

Ensayado sobre la copia fría posterior a AD197, con lo de main (AD198 a AD208,
CT179, CT180 y CT183), Personal37 y Personal38. Repetirla se para en la
preimagen. La búsqueda de la asignación usa
`asignacion_perfil_principal_perfil` y las claves primarias; con pocas filas el
planificador prefiere recorrer la tabla.
