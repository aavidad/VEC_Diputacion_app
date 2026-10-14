# Dietas: preparación de la candidata

Encargo de dirección del 14 de octubre de 2026. Solo se trabaja en la copia
local desechable. Dirección conserva la instalación de cidonia.

Estado: **en preparación**. No instalar este directorio como candidata completa.
La liquidación económica, la subida de justificantes y sus documentos aún no
se han recorrido de punta a punta.

## Código que se reutiliza

- Borradores y documento de comisión, cálculo y circuito del módulo Dietas.
- Relación y adscripción que proyecta Personal; la escritura completa de la
  asignación sigue cerrada.
- Perfiles centrales para acreditar etapa y unidad de cada revisor.
- Documentos para análisis, custodia y descarga; los justificantes declarados
  por referencia no se consideran ficheros custodiados.

El inventario contrastó las ramas antiguas de continuación, justificantes y
documento liquidado. Ninguna completa la liquidación nominal ni su descarga.
Se conservan sus contratos útiles; no se incorporan sus paquetes de datos de
presentación al portal.

## Preparación y activación

1. Restaurar el respaldo HZ37 en PostgreSQL 18 propio y desechable, incluidos
   globales y permisos de base y tipos. Conservar los originales del respaldo.
2. Inventariar funciones, huellas, permisos, asignaciones y vigencias antes de
   instalar SQL. Usar [lista_sql.txt](lista_sql.txt); nunca reaplicar una
   migración instalada ni ejecutar una reversión.
3. Preparar los perfiles ficticios con ámbito y vigencia de al menos sesenta
   días. Dirección autorizó el 14/10 a recrear sus datos directamente por SQL
   en la copia propia; no hace falta conservar la retirada R1D. El antiguo
   paquete F4b sigue sin encajar con AUT22 y no se cambia esa guarda. Para la
   instalación final, usar las vías de Administración que documentará H.
4. Preparar fuera de Git el material de identidad y las once conexiones
   separadas que describe [03_entorno.md](../03_entorno.md), sección «Dietas
   con la composición actual». Exigir TLS verificado y SCRAM, con rechazo de
   conexiones sin TLS.
5. Configurar las cinco variables `VEC_OSRM_*` del
   [despliegue cartográfico](../../osrm-granada/README.md), la doble llave de
   desarrollo y `VEC_DEVELOPMENT_MATERIAL_DIR`.
6. Añadir `dietas` a la lista existente `VEC_PORTAL_MODULOS_VISIBLES`, sin
   sustituir los demás módulos. Activar `VEC_DIETAS_BORRADORES_ENABLED=true`.
7. Con el entorno definitivo, ejecutar `vec-server comprobar-dietas` y después
   arrancar el mismo binario y material. Si la comprobación falla, resolver su
   causa antes de continuar el recorrido.
8. Arrancar también sin la configuración de Dietas y comprobar sesión, salud
   y portal. Repetir con una configuración incompleta: Dietas debe quedar
   cerrada y las capacidades comunes deben seguir funcionando.

No hay nuevas variables de configuración en los primeros cortes de arranque
y recuperación de bandejas. Las conexiones y el material existentes mantienen
sus comprobaciones; el aislamiento no los convierte en opcionales para usar
Dietas.

## Recorrido y comprobaciones

El resultado observado está en [recorrido.md](recorrido.md). El recorrido
Bolsa → Contratación temporal de `docs/producto/FLUJO_APP.md`, apartado 4.0,
conserva su caso de aceptación y no se modifica desde esta entrega.

Una liquidación conservará el documento y catálogo exactos, líneas, motivos,
total, versión, recibo, auditoría e historia. PDF y Word saldrán de esa
instantánea y quedarán pendientes de firma hasta conectar su circuito. La
fiscalización y la liquidación nunca se presentan como pago.
