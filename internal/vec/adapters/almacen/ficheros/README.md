# Almacén local de originales

El directorio raíz y sus subdirectorios deben pertenecer al uid del proceso y
ser privados. Al iniciar, el adaptador toma el cerrojo exclusivo y elimina los
ficheros regulares `tmp_*` que quedaron tras un corte; solo
registra la cantidad eliminada. Los escritores mantienen el cerrojo compartido
`.volcados` hasta concluir la operación y no se limpian durante otro arranque.
La configuración
`MaximoVolcadosConcurrentes` limita los volcados simultáneos por instancia;
su valor predeterminado es 2 y se valida entre 1 y 16 cuando se fija. El
semáforo acota las copias de contenido en curso, no los temporales vivos: el
volcado lo libera al terminar y el temporal ya verificado sigue existiendo
mientras la escritura espera el cerrojo exclusivo para materializarlo o
descartarlo. En el peor caso pueden coexistir tantos temporales como
escrituras concurrentes haya en espera, cada uno de hasta `TamanoMaximo`
bytes; el volumen debe dimensionarse para ello.

Este adaptador compone rutas con `filepath.Join`. No emplea descriptores de
directorio con `openat` ni `RESOLVE_BENEATH`: un proceso con el mismo uid puede
sustituir un subdirectorio entre la validación inicial y una operación. El
aislamiento del uid y del volumen privado forma parte del modelo de amenaza.
