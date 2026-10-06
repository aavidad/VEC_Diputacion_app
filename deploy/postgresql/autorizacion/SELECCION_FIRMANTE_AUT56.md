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
    con el control del rol habilitado y con exactamente dos ámbitos: la
    organización y la unidad del paso, cada una con un solo valor.

  Devuelve la selección (perfil activo, rol, cargo y enlace), la cuenta y el
  vínculo del certificado, y las versiones y huellas de la asignación, el rol
  y el control. No escribe ni concede nada. Ninguna o más de una asignación se
  deniegan. La lectura no deja auditoría propia: el adaptador la hace en una
  transacción que siempre deshace, y lo que se audita es el consumo de la
  decisión al firmar.
- **La vía directa de CT172 queda cerrada para el ejecutor.** Con el enlace por
  tipo, quien llamase a `registrar_firma_verificada_v2` podría elegir el tipo
  de recurso. AUT56 le retira el EXECUTE (lo hace como propietario CT) y
  comprueba al final que ni la v2 ni, si existe, la v3 son ejecutables por él.
  Toda firma entra por CT176, que ata tipo, acción, finalidad, cargo y enlace al
  plan publicado.

El vector `pruebas_sql/aut56_seleccion_firmante.sql` da 15/15. Antes de la
asignación buena siembra siete que fallan en una sola condición cada una
(caducada, revocada, otra unidad, sin unidad, una dimensión de más, dos valores
en la unidad y rol con control retirado): si se quitase cualquiera de esos
filtros, la selección dejaría de denegarse. Comprueba además:
- selección positiva, con todos los campos;
- certificado ajeno, organización distinta de la del certificado, enlace por
  documento y otro rol: denegados con su mensaje;
- dos asignaciones buenas: ambigua;
- LOGIN que mezcla ejecutor CT y propietario AUT, y LOGIN ajeno sin EXECUTE;
- vía directa de CT172 cerrada para el ejecutor;
- AUT35 por tipo;
- con el mismo cargo, una decisión para el original A presentada con el
  material del original B: CT172 (v2, o v3 si CT181 está instalada) la rechaza
  antes del consumidor V3, mientras que con A sí llega a él.

Ensayado sobre la copia fría posterior a AD197, con lo de main (AD198 a AD208,
CT179, CT180 y CT183), Personal37 y Personal38, en tres órdenes: sin AD206 ni
CT181, con ellas antes de AUT56 y con ellas después. Repetirla se para en la
preimagen. La búsqueda de la asignación usa
`asignacion_perfil_principal_perfil` y las claves primarias; con pocas filas el
planificador prefiere recorrer la tabla.
