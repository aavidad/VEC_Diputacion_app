package application

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaCotejoRequerida = errors.New("vec: dependencia de cotejo requerida")
	ErrOrdenCotejoInvalida        = errors.New("vec: orden de cotejo invalida")
	ErrResultadoCotejoInvalido    = errors.New("vec: resultado de cotejo invalido")
	ErrCotejoNoDisponible         = errors.New("vec: cotejo no disponible")
)

const (
	moduloNucleoDocumental         = "documentos"
	vigenciaReservaOperacion       = 5 * time.Minute
	minimoEntropiaCotejoAplicacion = 128

	AccionCrearBorradorPoliticaCotejo      = "vec.documentos.cotejo.politicas.crear"
	AccionActualizarBorradorPoliticaCotejo = "vec.documentos.cotejo.politicas.actualizar"
	AccionPublicarPoliticaCotejo           = "vec.documentos.cotejo.politicas.publicar"
	AccionRetirarPoliticaCotejo            = "vec.documentos.cotejo.politicas.retirar"
	AccionReservarCodigoCotejo             = "vec.documentos.cotejo.codigos.reservar"
	AccionActivarCodigoCotejo              = "vec.documentos.cotejo.codigos.activar"
	AccionRetirarCodigoCotejo              = "vec.documentos.cotejo.codigos.retirar"
	AccionSustituirCodigoCotejo            = "vec.documentos.cotejo.codigos.sustituir"
	AccionConsultaPublicaCotejo            = "vec.documentos.cotejo.consultar_publico"
	AccionConsultaProtegidaCotejo          = "vec.documentos.cotejo.consultar_protegido"
	AccionRevisionInternaCotejo            = "vec.documentos.cotejo.revisar"
)

// ServicioCotejo coordina gobierno, emision y consulta sin conocer el motor de
// base de datos, el KMS/Vault, el generador QR, la firma ni el registro.
type ServicioCotejo struct {
	politicas         ports.CatalogoPoliticasCotejo
	gobiernoPoliticas ports.RepositorioGobiernoPoliticasCotejo
	codigos           ports.RepositorioCodigosCotejo
	documentos        ports.RepositorioDocumentosLogicos
	autorizador       ports.Autorizador
	generadorValor    ports.GeneradorValorCodigoCotejo
	generadorID       ports.GeneradorIDCodigoCotejo
	selladorIndice    ports.SelladorIndiceCodigoCotejo
	selladorSolicitud ports.SelladorSolicitudCotejo
	protector         ports.ProtectorCodigoCotejo
	evidenciasEmision ports.FuenteEvidenciaEmisionDocumento
	reloj             ports.Reloj
}

func NuevoServicioCotejo(
	politicas ports.CatalogoPoliticasCotejo,
	gobiernoPoliticas ports.RepositorioGobiernoPoliticasCotejo,
	codigos ports.RepositorioCodigosCotejo,
	documentos ports.RepositorioDocumentosLogicos,
	autorizador ports.Autorizador,
	generadorValor ports.GeneradorValorCodigoCotejo,
	generadorID ports.GeneradorIDCodigoCotejo,
	selladorIndice ports.SelladorIndiceCodigoCotejo,
	selladorSolicitud ports.SelladorSolicitudCotejo,
	protector ports.ProtectorCodigoCotejo,
	evidenciasEmision ports.FuenteEvidenciaEmisionDocumento,
	reloj ports.Reloj,
) (*ServicioCotejo, error) {
	if politicas == nil || gobiernoPoliticas == nil || codigos == nil || documentos == nil ||
		autorizador == nil || generadorValor == nil || generadorID == nil || selladorIndice == nil ||
		selladorSolicitud == nil || protector == nil || evidenciasEmision == nil || reloj == nil {
		return nil, ErrDependenciaCotejoRequerida
	}
	return &ServicioCotejo{
		politicas:         politicas,
		gobiernoPoliticas: gobiernoPoliticas,
		codigos:           codigos,
		documentos:        documentos,
		autorizador:       autorizador,
		generadorValor:    generadorValor,
		generadorID:       generadorID,
		selladorIndice:    selladorIndice,
		selladorSolicitud: selladorSolicitud,
		protector:         protector,
		evidenciasEmision: evidenciasEmision,
		reloj:             reloj,
	}, nil
}

func validarContextoCotejo(
	principal domain.Principal,
	perfilActivo, finalidad, motivo, correlacionRef string,
	garantiaMinima domain.AuthAssurance,
) error {
	if err := principal.Validate(); err != nil {
		return errors.Join(ErrOrdenCotejoInvalida, err)
	}
	if strings.TrimSpace(perfilActivo) == "" || strings.TrimSpace(finalidad) == "" ||
		strings.TrimSpace(motivo) == "" || strings.TrimSpace(correlacionRef) == "" {
		return ErrOrdenCotejoInvalida
	}
	if !principal.AuthAssurance.Cumple(garantiaMinima) {
		return domain.ErrGarantiaInsuficiente
	}
	return nil
}

func recursoPoliticaCotejo(politica domain.PoliticaCotejo) domain.RecursoAutorizable {
	return domain.RecursoAutorizable{
		Referencia: politica.Referencia(),
		ModuloID:   moduloNucleoDocumental,
		Tipo:       "politica_cotejo",
		Atributos: map[string]string{
			"estado":       string(politica.Estado),
			"clase_acceso": string(politica.ClaseAcceso),
		},
	}
}

func referenciaDocumentoCotejo(referencia domain.ReferenciaDocumento) string {
	return referencia.ID + ":" + strconv.Itoa(referencia.Version)
}
