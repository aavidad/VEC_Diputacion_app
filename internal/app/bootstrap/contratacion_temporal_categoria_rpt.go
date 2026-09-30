package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// El vínculo procede de la historia propia del expediente o de una
// correspondencia gobernada expresa. Una categoría aislada del análisis no
// permite reconstruir la publicación ni el recibo de reserva que faltan.
type vinculoContinuidadCategoriaRPT struct {
	organizacionRef   string
	expedienteRef     string
	versionExpediente uint64
	categoriaRef      string
	publicacion       puertosvec.ReferenciaPublicacionRPT
	usoRef            string
	reservaReciboRef  string
}

type fuenteVinculoContinuidadCategoriaRPT interface {
	ResolverVinculoContinuidadCategoriaRPT(context.Context, ports.ExpedienteParaSeleccion) (vinculoContinuidadCategoriaRPT, error)
}

// Cada orden debe resolver actor/perfil desde la frontera confiable y obtener
// una decisión V3 nueva. El lector común coteja su material y consume esa
// decisión junto con la auditoría en la misma transacción.
type autoridadLecturasContinuidadCategoriaRPT interface {
	AutorizarConsultaUsoCategoriaRPT(context.Context, puertosvec.ConsultaUsoCategoriaRPT) (puertosvec.OrdenUsoCategoriaRPT, error)
	AutorizarPublicacionCategoriaRPT(context.Context, puertosvec.ConsultaPublicacionCategoriaRPT) (puertosvec.OrdenPublicacionCategoriaRPT, error)
}

type consumidorContinuidadCategoriaRPT struct {
	descriptor puertosvec.DescriptorCatalogoRPT
	lector     puertosvec.LectorCategoriasRPT
	fuente     fuenteVinculoContinuidadCategoriaRPT
	autoridad  autoridadLecturasContinuidadCategoriaRPT
}

// La composición instala todas las dependencias; ninguna procede de HTTP.
// Este consumidor no crea reservas: Personal necesita su propio uso nuevo.
func nuevoConsumidorContinuidadCategoriaRPT(descriptor puertosvec.DescriptorCatalogoRPT,
	lector puertosvec.LectorCategoriasRPT, fuente fuenteVinculoContinuidadCategoriaRPT,
	autoridad autoridadLecturasContinuidadCategoriaRPT,
) (*consumidorContinuidadCategoriaRPT, error) {
	if descriptor.CatalogoID == "" || descriptor.ModuloID == "" ||
		dependenciaEsNulaContratacionTemporalDesarrollo(lector) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(fuente) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(autoridad) {
		return nil, puertosvec.ErrLecturaRPTNoDisponible
	}
	return &consumidorContinuidadCategoriaRPT{descriptor: descriptor, lector: lector, fuente: fuente, autoridad: autoridad}, nil
}

// validar consulta el uso admitido y su publicación original. El control
// actual no cambia esa publicación: deshabilitar no impide concluir el uso.
// El resultado es una referencia de datos; no concede el permiso de Bolsa.
func (c *consumidorContinuidadCategoriaRPT) validar(ctx context.Context, expediente ports.ExpedienteParaSeleccion) (string, error) {
	if contextoInterfazNulo(ctx) || c == nil || dependenciaEsNulaContratacionTemporalDesarrollo(c.lector) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.fuente) || dependenciaEsNulaContratacionTemporalDesarrollo(c.autoridad) || expediente.Fiscalizado.Analisis == nil ||
		expediente.Fiscalizado.Validar() != nil {
		return "", puertosvec.ErrLecturaRPTInvalida
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// El resolvedor no comparte punteros con el análisis que se coteja después.
	copia := expediente
	copia.Fiscalizado = expediente.Fiscalizado.Clonar()
	v, err := c.fuente.ResolverVinculoContinuidadCategoriaRPT(ctx, copia)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if v.organizacionRef != expediente.Fiscalizado.OrganizacionRef || v.expedienteRef != expediente.Fiscalizado.Referencia ||
		v.versionExpediente != expediente.Fiscalizado.Version ||
		v.categoriaRef != expediente.Fiscalizado.Analisis.CategoriaRef || v.categoriaRef == "" ||
		v.publicacion.CatalogoID != c.descriptor.CatalogoID || v.publicacion.Version < 1 ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(v.publicacion.HuellaSHA256) ||
		len(v.usoRef) < 3 || len(v.usoRef) > 160 || len(v.reservaReciboRef) < 3 || len(v.reservaReciboRef) > 160 {
		return "", puertosvec.ErrLecturaRPTNoConfiable
	}
	consultaUso := puertosvec.ConsultaUsoCategoriaRPT{Consumidor: "contratacion_temporal", UsoRef: v.usoRef, ReservaReciboRef: v.reservaReciboRef}
	ordenUso, err := c.autoridad.AutorizarConsultaUsoCategoriaRPT(ctx, consultaUso)
	if err != nil {
		return "", err
	}
	if ordenUso.Consulta != consultaUso {
		return "", puertosvec.ErrLecturaRPTDenegada
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	uso, err := c.lector.ConsultarUsoCategoriaRPT(ctx, ordenUso)
	if err != nil {
		return "", err
	}
	if !uso.Encontrado || uso.Uso == nil || !evidenciaContinuidadCategoriaRPTValida(uso.Evidencia, ordenUso.Autorizacion) ||
		uso.Uso.Consumidor != consultaUso.Consumidor || uso.Uso.UsoRef != consultaUso.UsoRef ||
		uso.Uso.ReservaReciboRef != consultaUso.ReservaReciboRef || uso.Uso.CategoriaID != v.categoriaRef ||
		uso.Uso.Publicacion != v.publicacion || !usoContinuidadCategoriaRPTValido(*uso.Uso) {
		return "", puertosvec.ErrLecturaRPTNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	consultaPublicacion := puertosvec.ConsultaPublicacionCategoriaRPT{Referencia: v.publicacion, CategoriaID: v.categoriaRef}
	ordenPublicacion, err := c.autoridad.AutorizarPublicacionCategoriaRPT(ctx, consultaPublicacion)
	if err != nil {
		return "", err
	}
	if ordenPublicacion.Consulta != consultaPublicacion ||
		ordenPublicacion.Autorizacion.ResumenCapacidad().DecisionRef() == ordenUso.Autorizacion.ResumenCapacidad().DecisionRef() {
		return "", puertosvec.ErrLecturaRPTDenegada
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	publicacion, err := c.lector.LeerPublicacionCategoriaRPT(ctx, ordenPublicacion)
	if err != nil {
		return "", err
	}
	if !publicacion.Encontrado || publicacion.Publicacion == nil || publicacion.Entrada == nil ||
		!evidenciaContinuidadCategoriaRPTValida(publicacion.Evidencia, ordenPublicacion.Autorizacion) ||
		publicacion.Publicacion.Referencia != v.publicacion || publicacion.Entrada.Clave != v.categoriaRef {
		return "", puertosvec.ErrLecturaRPTNoConfiable
	}
	suma := sha256.Sum256([]byte(publicacion.Publicacion.DocumentoCanonico))
	if hex.EncodeToString(suma[:]) != v.publicacion.HuellaSHA256 {
		return "", puertosvec.ErrLecturaRPTNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return v.categoriaRef, nil
}

func usoContinuidadCategoriaRPTValido(u puertosvec.UsoCategoriaRPT) bool {
	if !instanteContinuidadCategoriaRPTValido(u.ReservadoEn) {
		return false
	}
	switch u.Estado {
	case "reservado":
		return u.Revision == 1 && u.TerminalReciboRef == nil && u.TerminalEn == nil
	case "confirmado":
		return u.Revision == 2 && u.TerminalReciboRef != nil && *u.TerminalReciboRef != "" && u.TerminalEn != nil &&
			instanteContinuidadCategoriaRPTValido(*u.TerminalEn) && !u.TerminalEn.Before(u.ReservadoEn)
	default:
		return false
	}
}

func evidenciaContinuidadCategoriaRPTValida(e puertosvec.EvidenciaLecturaRPT, a puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	r := a.ResumenCapacidad()
	return e.ConsumoNuevo && e.DecisionRef == r.DecisionRef() && e.EfectoRef == r.EfectoRef() &&
		e.HuellaEfectoSHA256 == r.EfectoHuellaSHA256() &&
		huellaSHA256ValidaContratacionTemporalDesarrollo(e.ConsumoHuellaSHA256) && e.AuditoriaRef != "" &&
		instanteContinuidadCategoriaRPTValido(e.ConsumidaEn) && !e.ConsumidaEn.Before(r.EmitidaEn()) && e.ConsumidaEn.Before(r.ExpiraEn())
}

func instanteContinuidadCategoriaRPTValido(t time.Time) bool {
	_, desfase := t.Zone()
	return !t.IsZero() && desfase == 0 && t.Nanosecond()%1000 == 0
}
