# Recorrido de desarrollo en una orden — F1-131

Desde una revisión limpia, la orden crea una copia propia con certificados,
claves, conexiones y dos administradores nuevos. Solo lee el volcado HZ37 y
sus dos anexos de permisos; no ejecuta recetas privadas de otros equipos.

```sh
export VEC_COPIA_ORIGEN=/ruta/privada/de/la/referencia
python3 deploy/principal/copia_local.py crear \
  --equipo l --instancia recorrido --recorrido-completo \
  --puerto-pg 55531 --puerto-app 18531 --puerto-admin 19531 \
  --puerto-smtp 11531 --puerto-verificador 19831 \
  --binario-verificador /ruta/al/software/grxfirma \
  --sha-verificador HUELLA_SHA256_DEL_BINARIO \
  --binario-afirmauri /ruta/al/software/grxfirma-afirmauri \
  --sha-afirmauri HUELLA_SHA256_DEL_MANEJADOR
```

El software se declara expresamente. La fábrica comprueba su huella y lo
copia con permisos privados antes de restaurar PostgreSQL. Rechaza GrxFirma
anterior a 0.0.106 y comprueba que acepta el modo de solo verificación.
Una compilación que declara «dev» requiere su huella expresa; se registra
esa versión real. No descarga ni modifica el software instalado.

Las dos opciones de `afirmauri` se proporcionan juntas. Este manejador abre
las solicitudes de firma de Chrome; es un ejecutable distinto del verificador.
La fábrica coteja su huella, comprueba que se identifica como
`grxfirma-afirmauri` y conserva una copia privada con su versión real.
La sonda no inicia una firma ni un servicio. El manejador es opcional para
crear la base y arrancar la aplicación; su ausencia queda reflejada en el
estado y no acredita la firma desde Chrome. La receta no modifica el
manejador de protocolos, las claves ni la confianza del usuario del sistema.

Se necesitan Docker local con PostgreSQL 18.4 y Mailpit, Go y sus dependencias
ya disponibles, y al menos 34 GiB de memoria disponible. Los cinco puertos
deben estar libres y ser distintos. Las carpetas comunes `/dev/shm/vec` y
`/dev/shm/vec/pg` deben tener permisos 0700 o 0755; la fábrica crea las que
faltan y rechaza otras condiciones sin cambiar permisos compartidos.

La secuencia restaura la referencia y sus permisos, registra el identificador
de PostgreSQL y aplica la lista SQL indicada. Prepara Bolsa, el aspirante,
dos técnicos de RRHH y los dos actores del centro seleccionado en el catálogo.
Compila la aplicación y Administración desde la misma revisión.

Después confirma fuentes, unidad y dos cuentas de Administración con sus
recibos nativos. Admite los catálogos de Organización histórica y origen de
Bolsa por la herramienta existente. Dos administradores proponen y aprueban
los roles simples y el plan. Tras publicar el rol, la fábrica llama a la
orden existente que activa la consulta de Organización histórica. Los roles compuestos de los cuatro firmantes son
**semilla técnica de copia de desarrollo, no gobierno**. Su publicación
gobernada corresponde a F1-133.

La etapa nominal confirma catorce actos, el plan y cuatro admisiones. La
configuración de Documentos usa el sujeto del certificado RRHH reconocido
por su resolvedor, separado de la referencia de Persona. La custodia conserva
los perfiles nominales de los cuatro firmantes. Personal usa las referencias
de la identidad propia y la unidad confirmada. Los empleados de los cuatro
firmantes se preparan con el guion de datos versionado existente.

Los certificados de desarrollo pueden cubrir 400 días. Cargos, admisiones,
asignaciones y autoridades nuevos cubren de 60 a 90 días. La ventana fija
heredada de ciertos perfiles del arranque está comunicada a dirección;
esta receta no cambia esa lógica de negocio.

Cada etapa conserva materiales y recibos privados. Ante una respuesta
incierta se concilia el acto original. Si cambia la revisión o la composición
de perfiles, crea otra instancia: no se admite actualizar en su sitio una
copia que ya consumió una composición distinta.

```sh
python3 deploy/principal/copia_local.py reiniciar --equipo l --instancia recorrido
python3 deploy/principal/copia_local.py comprobar --equipo l --instancia recorrido
python3 deploy/principal/copia_local.py borrar --equipo l --instancia recorrido
```

El registro de fábrica acredita las etapas confirmadas, no una firma jurídica
ni el recorrido completo del expediente. La aceptación de esta orden sigue
pendiente de copia nueva integrada, sondas tras reinicio y revisión
independiente. El caso de aceptación es FLUJO_APP.md, apartado 4.0.
