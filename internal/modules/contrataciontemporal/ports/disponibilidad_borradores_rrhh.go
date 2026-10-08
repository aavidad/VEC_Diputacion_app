package ports

// EstadoDisponibilidadBorradoresRRHH describe únicamente el montaje de la
// consulta documental. La ruta documental vuelve a autorizar cada petición.
type EstadoDisponibilidadBorradoresRRHH string

const (
	BorradoresRRHHSinMontaje   EstadoDisponibilidadBorradoresRRHH = "sin_montaje"
	BorradoresRRHHIndisponible EstadoDisponibilidadBorradoresRRHH = "indisponible"
)
