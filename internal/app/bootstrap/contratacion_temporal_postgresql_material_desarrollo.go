package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	gocose "github.com/veraison/go-cose"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type firmanteAtestacionAltaContratacionTemporalDesarrollo struct {
	claveID string
	privada ed25519.PrivateKey
	reloj   relojContratacionTemporalDesarrollo
}

func (f *firmanteAtestacionAltaContratacionTemporalDesarrollo) FirmarAtestacionAutorizacionV3(
	ctx context.Context,
	solicitud puertosvec.SolicitudFirmaAtestacionAutorizacionV3,
) (puertosvec.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if ctx == nil || f == nil || len(f.privada) != ed25519.PrivateKeySize {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	cabecera, err := solicitud.Cabecera()
	if err != nil || cabecera.ClaveID != f.claveID ||
		cabecera.Audiencia != audienciaAtestacionContratacionTemporalDesarrollo {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	defer borrarBytes(mensaje)
	aad, err := confianzaatestacion.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	sobre := gocose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	sobre.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(f.claveID)
	sobre.Payload = append([]byte(nil), mensaje...)
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, f.privada)
	if err != nil || sobre.Sign(rand.Reader, aad, firmante) != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	sobre.Payload = nil
	sobre.Headers.RawProtected = nil
	sobre.Headers.RawUnprotected = nil
	firma, err := sobre.MarshalCBOR()
	if err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	defer borrarBytes(firma)
	huella := sha256.Sum256(mensaje)
	return puertosvec.NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud, firma, "evidencia:firma:ct:desarrollo:"+hex.EncodeToString(huella[:8]),
		f.reloj.Ahora(),
	)
}

type proveedorMaterialAltaContratacionTemporalDesarrollo struct {
	fuenteConfianza *fuenteConfianzaRenovableCTDesarrollo
	atestador       *aplicacionvec.ServicioAtestacionesAutorizacionV3
	confianza       *confianzaatestacion.ServicioConfianzaAtestacionAutorizacionV3
	emisor          *confianzaatestacion.EmisorCapacidadesAtestacionAutorizacionV3
	raiz            confianzaatestacion.RaizPublicaAtestacionAutorizacionV3
	contexto        dominiovec.ResultadoContextoActorRegistradoV2
	motivo          dominiovec.ReferenciaEntradaCatalogo
}

func nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(
	material materialAtestacionContratacionTemporalDesarrollo,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	if soporte == nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	firmante := &firmanteAtestacionAltaContratacionTemporalDesarrollo{
		claveID: material.claveID,
		privada: append(ed25519.PrivateKey(nil), material.privada...),
		reloj:   reloj,
	}
	atestador, err := aplicacionvec.NuevoServicioAtestacionesAutorizacionV3(
		dominiovec.CabeceraAtestacionAutorizacionV3{
			FormatoVersion: dominiovec.VersionFormatoAtestacionAutorizacionV3,
			Suite:          confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
			ClaveID:        material.claveID,
			Audiencia:      audienciaAtestacionContratacionTemporalDesarrollo,
		},
		firmante,
	)
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	confianza, err := confianzaatestacion.NuevoServicioConfianzaAtestacionAutorizacionV3(
		material.configuracion, reloj,
	)
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	emisor, err := confianzaatestacion.NuevoEmisorCapacidadesAtestacionAutorizacionV3(
		material.capacidad, reloj,
	)
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	contexto, err := soporte.contexto.Resultado.Clonar()
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return &proveedorMaterialAltaContratacionTemporalDesarrollo{
		atestador: atestador, confianza: confianza, emisor: emisor, fuenteConfianza: material.fuenteConfianza,
		raiz: material.raiz, contexto: contexto, motivo: soporte.motivo,
	}, nil
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) ProveerMaterialConfirmacionAlta(
	ctx context.Context,
	orden ports.OrdenConfirmarAltaCandidata,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := orden.Datos()
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ports.ErrPersistenciaNoDisponible
	}
	return p.proveerMaterialConfirmacion(
		ctx,
		datos.SolicitudAutorizacionV3,
		datos.DecisionAutorizacionV3,
		datos.ConfirmacionRegistroV3,
		p.motivo,
		p.contexto,
	)
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) ProveerMaterialConfirmacionAsignacion(
	ctx context.Context,
	orden ports.OrdenConfirmarAsignacion,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := orden.Datos()
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	material, err := p.proveerMaterialConfirmacion(
		ctx,
		datos.SolicitudV3,
		datos.DecisionV3,
		datos.ConfirmacionV3,
		datos.Politica.MotivoAutorizacion,
		datos.ContextoAutorizacion.Resultado,
	)
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	return material, nil
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) ProveerMaterialConfirmacionInformeJuridico(
	ctx context.Context,
	orden ports.OrdenConfirmarInformeJuridico,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	material, err := p.proveerMaterialConfirmacion(
		ctx,
		orden.Evidencia.SolicitudV3,
		orden.Evidencia.DecisionV3,
		orden.Evidencia.ConfirmacionV3,
		orden.Configuracion.MotivoAutorizacion,
		orden.Evidencia.Contexto.Resultado,
	)
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	return material, nil
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) ProveerMaterialConfirmacionFiscalizacion(
	ctx context.Context,
	orden ports.OrdenConfirmarFiscalizacion,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	material, err := p.proveerMaterialConfirmacion(
		ctx,
		orden.Evidencia.SolicitudV3,
		orden.Evidencia.DecisionV3,
		orden.Evidencia.ConfirmacionV3,
		orden.Politica.MotivoAutorizacion,
		orden.Evidencia.Contexto.Resultado,
	)
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return material, nil
}

// ProveerMaterialConfirmacionSubsanacionReparo entrega al confirmador sólo el
// material atestado de la autorización ya ligada. No reutiliza la capacidad de
// alta como autorización: conserva el motivo y contexto que validó el caso de
// uso en la misma operación durable.
func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) ProveerMaterialConfirmacionSubsanacionReparo(
	ctx context.Context,
	orden ports.OrdenConfirmarSubsanacionReparo,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	material, err := p.proveerMaterialConfirmacion(
		ctx,
		orden.Evidencia.SolicitudV3,
		orden.Evidencia.DecisionV3,
		orden.Evidencia.ConfirmacionV3,
		orden.Politica.MotivoAutorizacion,
		orden.Evidencia.Contexto.Resultado,
	)
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return material, nil
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) proveerMaterialConfirmacion(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	contextoOriginal dominiovec.ResultadoContextoActorRegistradoV2,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if ctx == nil || p == nil || p.atestador == nil || p.confianza == nil || p.emisor == nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	contexto, err := contextoOriginal.Clonar()
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	datosSolicitud, err := solicitud.Datos()
	ordenConcesion, errOrden := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud, decision, motivo, contexto,
	)
	if err != nil || errOrden != nil || datosSolicitud.ReferenciaMotivo != motivo ||
		datosSolicitud.VinculoAutenticacionActor.ValidarPara(contexto) != nil ||
		confirmacion.ValidarPara(ordenConcesion) != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	atestacion, err := p.atestador.Atestar(
		ctx, decision, motivo, contexto,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	prueba, err := p.Verificar(
		ctx, solicitud, decision, motivo, contexto, atestacion,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	capacidad, err := p.emisor.Emitir(
		ctx, solicitud, decision, motivo, contexto, atestacion, prueba,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	material, err := confianzaatestacion.NuevoMaterialConsumoAutorizacionAtestadaV3(
		solicitud, decision, motivo, contexto, atestacion, prueba, capacidad, p.raiz,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	return material.ExportarMaterialParaConsumidor()
}
