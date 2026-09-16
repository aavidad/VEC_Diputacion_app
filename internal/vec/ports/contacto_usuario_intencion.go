package ports

import "context"

// SolicitudRegistroContactoRecuperable conserva la identidad propia y añade
// la intención estable que el cliente debe reutilizar tras una respuesta perdida.
type SolicitudRegistroContactoRecuperable struct {
	Solicitud SolicitudRegistroContactoUsuario
	IntentRef string
}

// PreparacionRegistroContactoRecuperable liga el sobre de este intento y la
// huella HMAC de la petición original. Nunca transporta el correo claro.
type PreparacionRegistroContactoRecuperable struct {
	Registro       PreparacionRegistroContactoUsuario
	IntentRef      string
	HuellaPeticion string
}

type OrdenRegistroContactoRecuperable struct {
	Preparacion  PreparacionRegistroContactoRecuperable
	Autorizacion SolicitudAccesoContactoUsuario
}

// ResultadoRegistroContactoRecuperable separa el recibo original de la
// auditoría del intento actual. Recuperar no crea otra versión ni otro outbox.
type ResultadoRegistroContactoRecuperable struct {
	Recuperado                 bool
	ReciboOriginal             ReciboContactoUsuario
	AuditoriaIntento           EvidenciaAuditoriaCentralContactoUsuario
	ConsumoIntentoRef          string
	ConsumoIntentoHuellaSHA256 string
}

type RegistroContactoRecuperable interface {
	GuardarContactoRecuperable(context.Context, OrdenRegistroContactoRecuperable) (ResultadoRegistroContactoRecuperable, error)
}
