"""Listas con las que se compone el paquete de datos de ejemplo de VEC.

Solo contienen piezas sueltas (nombres de pila, apellidos frecuentes, municipios
de la provincia de Granada con su código postal y nombres de calle corrientes);
ninguna combinación procede de una persona real ni de una exportación de Convoca.
"""
from __future__ import annotations

# Nombres de pila frecuentes en España (listas del INE), con tildes correctas.
NOMBRES_MUJER = (
    "Carmen", "María", "Ana María", "Josefa", "Isabel", "Laura", "María Dolores", "María Pilar",
    "Cristina", "Marta", "María Ángeles", "Lucía", "Francisca", "Antonia", "Dolores", "Sara",
    "Paula", "Elena", "María Luisa", "Raquel", "Rosa María", "Pilar", "Concepción", "Manuela",
    "María Jesús", "Mercedes", "Julia", "Beatriz", "Nuria", "Silvia", "Irene", "Alba", "Rosario",
    "Patricia", "Juana", "Teresa", "Encarnación", "Andrea", "Rocío", "Mónica", "Alicia", "Rosa",
    "Sonia", "Sandra", "Marina", "Susana", "Yolanda", "Ángela", "Inmaculada", "Natalia",
    "Eva María", "Esther", "Noelia", "Claudia", "Verónica", "Amparo", "Carolina", "Eva", "Lorena",
    "Ana Isabel", "Miriam", "Inés", "Sofía", "Victoria", "Adriana", "Macarena", "Emilia", "Gloria",
    "Remedios", "Virginia", "Trinidad", "Esperanza", "Angustias", "Aurora", "Lidia", "Estefanía",
    "Fátima", "Belén", "Milagros", "Alejandra", "Carla", "Celia", "Clara", "Elisa", "Lourdes",
    "María José", "María Isabel", "María Teresa", "María Victoria", "Ana Belén", "Marta María",
    "Nieves", "Asunción", "Consuelo", "Elvira", "Leticia", "Olga", "Vanesa", "Tamara",
)
NOMBRES_HOMBRE = (
    "Antonio", "José", "Manuel", "Francisco", "David", "Juan", "José Antonio", "Javier", "Daniel",
    "José Luis", "Francisco Javier", "Jesús", "Carlos", "Alejandro", "Miguel", "José Manuel",
    "Rafael", "Pedro", "Miguel Ángel", "Ángel", "José María", "Pablo", "Fernando", "Sergio", "Luis",
    "Jorge", "Alberto", "Juan Carlos", "Juan José", "Álvaro", "Diego", "Adrián", "Juan Antonio",
    "Raúl", "Enrique", "Ramón", "Iván", "Rubén", "Óscar", "Vicente", "Andrés", "Joaquín",
    "Santiago", "Víctor", "Eduardo", "Mario", "Roberto", "Jaime", "Francisco José", "Marcos",
    "Ignacio", "Alfonso", "Hugo", "Salvador", "Ricardo", "Emilio", "Guillermo", "Gonzalo",
    "Mariano", "Gabriel", "Julio", "Tomás", "Agustín", "Martín", "Nicolás", "Cristóbal", "Gregorio",
    "Lorenzo", "Samuel", "Ismael", "Rodrigo", "Héctor", "Julián", "Felipe", "Sebastián", "Domingo",
    "Esteban", "Juan Manuel", "Luis Miguel", "José Ramón", "Arturo", "Borja", "Emilio José",
    "Fernando José", "Leandro", "Marcelino", "Moisés", "Nicolás Jesús", "Patricio", "Teodoro",
)

# Apellidos frecuentes en España y en Andalucía oriental.
APELLIDOS = (
    "García", "Rodríguez", "González", "Fernández", "López", "Martínez", "Sánchez", "Pérez",
    "Gómez", "Martín", "Jiménez", "Hernández", "Ruiz", "Díaz", "Moreno", "Muñoz", "Álvarez",
    "Romero", "Gutiérrez", "Alonso", "Navarro", "Torres", "Domínguez", "Ramos", "Vázquez",
    "Ramírez", "Gil", "Serrano", "Morales", "Molina", "Blanco", "Suárez", "Castro", "Ortega",
    "Delgado", "Ortiz", "Marín", "Rubio", "Núñez", "Medina", "Sanz", "Castillo", "Iglesias",
    "Cortés", "Garrido", "Santos", "Guerrero", "Lozano", "Cano", "Cruz", "Méndez", "Flores",
    "Prieto", "Herrera", "Peña", "León", "Márquez", "Cabrera", "Gallego", "Calvo", "Vidal",
    "Campos", "Reyes", "Vega", "Fuentes", "Carrasco", "Aguilar", "Caballero", "Nieto", "Vargas",
    "Pascual", "Herrero", "Hidalgo", "Montero", "Lorenzo", "Santiago", "Benítez", "Durán",
    "Ibáñez", "Arias", "Mora", "Ferrer", "Carmona", "Vicente", "Rojas", "Soto", "Crespo",
    "Román", "Pastor", "Velasco", "Parra", "Sáez", "Moya", "Bravo", "Rivera", "Gallardo",
    "Soler", "Rivas", "Pardo", "Espinosa", "Maldonado", "Palacios", "Robles", "Escobar",
    "Sierra", "Valero", "Contreras", "Mesa", "Rosales", "Heredia", "Ávila", "Olmedo", "Jurado",
    "Pineda", "Quesada", "Padilla", "Salazar", "Morillas", "Linares", "Aguilera", "Castaño",
    "Villegas", "Montes", "Ruano", "Valverde", "Toledo", "Estévez", "Mingorance", "Puertas",
    "Olmo", "Zamora", "Rueda", "Barrios", "Tejada", "Sola", "Ferrón", "Maroto", "Cuadros",
    "Lara", "Bueno", "Casas", "Plaza", "Luque", "Ramiro", "Gámez", "Béjar", "Cobos", "Porcel",
)

# Municipios reales de la provincia de Granada con un código postal de su término.
MUNICIPIOS = (
    ("Granada", ("18001", "18002", "18003", "18004", "18005", "18006", "18007", "18008", "18009",
                 "18010", "18011", "18012", "18013", "18014", "18015")),
    ("Motril", ("18600",)), ("Almuñécar", ("18690",)), ("Armilla", ("18100",)),
    ("Maracena", ("18200",)), ("Baza", ("18800",)), ("Guadix", ("18500",)), ("Loja", ("18300",)),
    ("Santa Fe", ("18320",)), ("Las Gabias", ("18110",)), ("La Zubia", ("18140",)),
    ("Atarfe", ("18230",)), ("Huétor Vega", ("18198",)), ("Ogíjares", ("18151",)),
    ("Salobreña", ("18680",)), ("Íllora", ("18260",)), ("Alhama de Granada", ("18120",)),
    ("Albolote", ("18220",)), ("Peligros", ("18210",)), ("Cúllar Vega", ("18195",)),
    ("Churriana de la Vega", ("18194",)), ("Huéscar", ("18830",)), ("Órgiva", ("18400",)),
    ("Pinos Puente", ("18240",)), ("Montefrío", ("18270",)), ("Iznalloz", ("18550",)),
    ("Alfacar", ("18170",)), ("Otura", ("18630",)), ("Padul", ("18640",)), ("Dúrcal", ("18650",)),
)

VIAS = (
    "Calle Real", "Calle Nueva", "Calle Mayor", "Calle Ancha", "Calle Iglesia", "Calle Granada",
    "Calle San Sebastián", "Calle Sierra Nevada", "Calle Alhambra", "Calle Federico García Lorca",
    "Calle Jazmín", "Calle Olivo", "Calle Almendro", "Calle Rosales", "Calle Doctor Fleming",
    "Calle Reyes Católicos", "Calle Mariana Pineda", "Calle Andalucía", "Calle Molino",
    "Calle Huertas", "Calle Fuente", "Calle Horno", "Calle Cervantes", "Calle Antonio Machado",
    "Avenida de Andalucía", "Avenida de la Constitución", "Avenida de Madrid", "Avenida del Mar",
    "Plaza de la Constitución", "Plaza de España", "Camino de Ronda", "Carretera de la Sierra",
    "Paseo del Salón", "Calle Genil", "Calle Darro", "Calle Albaicín", "Calle Veleta",
    "Calle Mulhacén", "Calle Alpujarra", "Calle Vega",
)

# Méritos del detalle «con claves»: grupo, descripción del grupo y descripciones.
MERITOS_EXPERIENCIA = (
    "Servicios prestados en la Diputación de Granada en la misma categoría",
    "Servicios prestados en ayuntamientos de la provincia en la misma categoría",
    "Servicios prestados en otras administraciones públicas en puesto análogo",
    "Servicios prestados en empresa privada en puesto de igual contenido",
)
MERITOS_FORMACION = (
    "Curso de prevención de riesgos laborales (20 horas)",
    "Curso de protección de datos en la Administración local (30 horas)",
    "Curso de atención a la ciudadanía (25 horas)",
    "Curso de igualdad entre mujeres y hombres (20 horas)",
    "Curso de primeros auxilios (15 horas)",
    "Curso de administración electrónica (40 horas)",
    "Jornadas sobre el procedimiento administrativo común (10 horas)",
    "Titulación académica superior a la exigida",
    "Máster universitario relacionado con el puesto",
)
MOTIVOS_TRIBUNAL = (
    "No se acredita el número de horas",
    "Formación no relacionada con las funciones del puesto",
    "Periodo ya computado en otro apartado",
    "Certificado de servicios sin fecha de cese",
)

# Contratación temporal: causas de sustitución y observaciones de la unidad.
CAUSAS_SUSTITUCION = (
    "durante su situación de incapacidad temporal",
    "durante su permiso por nacimiento y cuidado de menor",
    "durante su periodo de vacaciones",
    "durante su excedencia por cuidado de familiares",
    "por la reducción de jornada que tiene concedida",
    "mientras ocupa temporalmente otro puesto por comisión de servicios",
)
OBSERVACIONES_SOLICITUD = (
    "La unidad no puede asumir las tareas con el personal disponible.",
    "Se solicita la incorporación antes del inicio del turno de mañana.",
    "El puesto atiende público de lunes a viernes.",
    "Se mantiene el horario habitual del puesto sustituido.",
    "El centro ha agotado la redistribución interna de tareas.",
    "La persona sustituida prevé reincorporarse al finalizar el periodo indicado.",
)
OBSERVACIONES_ANALISIS = (
    "Necesidad temporal acreditada con el parte de baja.",
    "Se comprueba la dotación presupuestaria del puesto.",
    "La jornada propuesta coincide con la del puesto sustituido.",
    "Sin incidencias en la documentación aportada por el centro.",
)
