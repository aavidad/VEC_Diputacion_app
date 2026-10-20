# Decisiones técnicas del estudio transversal de septiembre de 2026

Antecedente del corte `a1bc66b7`. Los acuerdos del estudio no acreditaron su
implementación ni una revisión exhaustiva del repositorio. Las reglas y el
estado actuales se consultan en [AGENTS.md](../../AGENTS.md),
[TRD](../producto/TRD.md) y [el plan](../producto/PLAN_IMPLEMENTACION.md).
La transcripción completa anterior permanece en la historia de Git.

## Arquitectura y simplificación

El estudio propuso concentrar cada operación en un caso de uso coordinador:
autorización vigente, carga, mutación, persistencia y auditoría transaccional,
con recibo. Los contratos y materiales con consumidor real se conservan;
retirar redundancias exige comprobar sus consumidores e invariantes.
La cancelación se transmite a entrada/salida y trabajo largo cancelable.

Se planteó una composición funcional común con fábricas de infraestructura
por entorno, contratos de puertos limitados a interfaces y datos de intercambio,
validación en dominio y serialización en adaptadores. Las proyecciones son
modelos de lectura de aplicación. La reducción se mide en el conjunto del
repositorio; mover ficheros o cumplir un número de líneas no acredita una mejora.

Las huellas y copias de datos requieren una finalidad de integridad o una
frontera mutable concreta. El artefacto de despliegue debe mantener coherentes
binario y web. Los privilegios se relacionan con una identidad técnica y un
consumidor vivo; las instalaciones nuevas necesitan una base reproducible sin
alterar las bases con historia.

## Errores, interfaz y pruebas

Los errores conservan su causa y dejan diagnóstico de frontera correlacionado,
con una respuesta pública mínima. El cliente comparte transporte con límites
de tiempo y tamaño, valida el contrato y evita duplicar reglas de negocio.
Los estados de pantalla se derivan de hitos reales; el número visible,
observaciones, catálogos, costes y documentos usan fuentes acreditadas.

Ningún adaptador nuevo se considera terminado sin consumidor y recorrido.
Las pruebas responden al comportamiento y al riesgo; las garantías de permisos,
identidad, criptografía y datos personales necesitan comprobaciones específicas.
Una inspección sin hallazgos no demuestra ausencia general de defectos.

## Propiedad de datos y límites externos

Bolsa conserva el orden, disponibilidad y llamamientos que consume Contratación.
Las reglas temporales, adjudicación y plazos se mantienen configurables mientras
falta confirmación funcional. Las personas candidatas acceden a su ficha con
identidad acreditada; la superficie pública ofrece solo campos aprobados.

Personal conserva una fuente de incorporación para los documentos y la ficha
de carga externa. Administración reutiliza identidad, autorización y auditoría
comunes en una superficie separada. Un certificado válido no concede acceso;
la asignación, ejercicio, revocación, denegación y consulta dejan constancia.
Los secretos de configuración requieren protección y lectura controlada.

La entrega distingue el registro local de una confirmación externa. Las copias
y restauración se comprueban antes de datos reales. El estudio identificó
integraciones y respuestas funcionales pendientes; no las dio por resueltas
mediante un acuerdo técnico ni acreditó el recorrido completo.
