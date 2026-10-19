"""Semilla de la publicación oficial de las bases de una copia local (F1-61, BC14).

Desde BC14 una convocatoria solo se abre a la inscripción si consta la
publicación oficial de sus bases. La copia usa datos inventados: esta semilla
declara una publicación ficticia para cada convocatoria que hoy tenga un plazo
de inscripción abierto, con fecha del día anterior a ese plazo. Se aplica en
cada arranque de la copia (`crear` y `reiniciar` pasan por `arrancar`), ya con
la lista SQL y el kit de datos aplicados, así que también recoge los plazos que
se abran después de crearla. Repetirla no escribe nada y, sin BC14, no hace
nada. Una versión publicada con material ilegible hace fallar el arranque de
forma visible, como fallarían la lista y la ficha de la aspirante.
"""
from pathlib import Path

SEMILLA = (Path(__file__).resolve().parent
           / 'f1_61_publicacion_oficial_bases' / 'declarar_ensayo_copia.sql')


def declarar(copia):
    """Aplica la semilla en la base de la copia, como su superusuario local."""
    copia.sql(SEMILLA, 'publicacion-bases.log')
