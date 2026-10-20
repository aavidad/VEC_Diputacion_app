# Ensayo aislado de Mi Bolsa y Dietas: antecedente del 23/09/2026

Referencia histórica. La operación vigente se consulta en
[FLUJO_APP](../producto/FLUJO_APP.md) y [el plan](../producto/PLAN_IMPLEMENTACION.md).
No se ejecutan sus procedimientos antiguos sobre el servidor congelado.

El ensayo usó PostgreSQL 18.4 y un volcado de esquema sin filas, conservando
propietarios, funciones, políticas RLS y ACL. Los roles se recrearon como
`NOLOGIN`; no se reprodujeron atributos de acceso ni membresías reales.
Por ello no acreditó identidad nominal, separación de funciones ni el
comportamiento con los datos del origen.

La preimagen contenía AD47, Bolsa16, AD48 y Bolsa17. Los incrementos estudiados
fueron Bolsa21 (relleno autorizado de vínculos), Bolsa22 (situación actual),
Bolsa23 (último llamamiento) y Dietas5 (auditoría de frontera).
Instalar la función de relleno no creó vínculos de candidatos: el relleno
necesitó datos acreditados y una operación autorizada independiente.

Se comprobaron precondiciones, instalación transaccional, evolución de
funciones, RLS forzosa y permisos de funciones y tablas. La comprobación de
`CONNECT` inspeccionó la concesión explícita de Dietas, sin inferirla de
`PUBLIC`. La ausencia de filas limitó la garantía de historia a estructura
y permisos; no acreditó persistencia de operaciones ni el recorrido web.

El aislamiento del contenedor y la retirada de temporales evitaron efectos en
el origen. No se ejecutó DOWN ni se reaplicaron migraciones con historia.
Las migraciones instaladas se conservan; cualquier corrección posterior y el
recorrido funcional requieren su propio ensayo y revisión de la cabeza final.
