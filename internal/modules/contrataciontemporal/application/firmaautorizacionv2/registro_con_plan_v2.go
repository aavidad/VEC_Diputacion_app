package firmaautorizacionv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RegistroConPlanV2 decora el registro usado por los servicios de firma.
// El lector anterior sólo se usa para consultas; nunca para escribir V2.
type RegistroConPlanV2 struct {
	fuente   ports.FuenteDescriptorPlanFijadoFirmaV2
	emisor   ports.EmisorMaterialPlanFirmaV2
	escritor ports.RegistradorFirmaConPlanV2
	consulta ports.RegistroFirmasVerificadasV2
}

var _ ports.RegistroFirmasVerificadasV2 = (*RegistroConPlanV2)(nil)

func NuevoRegistroConPlanV2(fuente ports.FuenteDescriptorPlanFijadoFirmaV2,
	emisor ports.EmisorMaterialPlanFirmaV2, escritor ports.RegistradorFirmaConPlanV2,
	consulta ports.RegistroFirmasVerificadasV2) (*RegistroConPlanV2, error) {
	if nuloRegistroPlan(fuente) || nuloRegistroPlan(emisor) || nuloRegistroPlan(escritor) || nuloRegistroPlan(consulta) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &RegistroConPlanV2{fuente, emisor, escritor, consulta}, nil
}

func (r *RegistroConPlanV2) RegistrarFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2,
	interior ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	if r == nil || ctx == nil || nuloRegistroPlan(r.fuente) || nuloRegistroPlan(r.emisor) || nuloRegistroPlan(r.escritor) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	m.EvidenciaFirmasCanonica = bytes.Clone(m.EvidenciaFirmasCanonica)
	defer clear(m.EvidenciaFirmasCanonica)
	if m.Validar() != nil || !domain.InstanteUTCCanonico(m.ComprobadaEn) ||
		m.PuestoFirmanteRef != "" || m.AmbitoFirmanteRef != "" || m.ActoCompetenciaRef != "" ||
		ValidarCapacidadFirmaVerificadaV2(interior, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	d, err := r.fuente.DescriptorPlanFijadoFirmaV2(ctx, m)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err != nil || d.Plan.Validar() != nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	seleccionado, err := CanonicoDescriptorFirmaVerificadaV2(m, d.Descriptor)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer clear(seleccionado)
	original, _ := interior.ExportarDescriptorParaConsumidor()
	defer clear(original)
	if !bytes.Equal(original, seleccionado) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	decision := interior.ExportarMaterialParaConsumidor().DecisionCanonica()
	defer clear(decision)
	huella := sha256.Sum256(decision)
	decisionSHA := hex.EncodeToString(huella[:])
	envoltorio, err := CanonicoPlanAutorizadoFirmaV2(m, d, decisionSHA)
	if err != nil {
		return cero, err
	}
	defer clear(envoltorio)
	recurso, err := RecursoPlanAutorizadoFirmaV2(m, d.Plan, decisionSHA, envoltorio)
	if err != nil {
		return cero, err
	}
	exterior, err := r.emisor.AutorizarMaterialPlanFirmaV2(ctx, m, recurso)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err != nil {
		return cero, err
	}
	bundle := ports.TransportarFirmaConPlanV2(interior, exterior, d.Plan, envoltorio, decisionSHA)
	if ValidarCapacidadFirmaConPlanV2(bundle, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	recibo, err := r.escritor.RegistrarFirmaConPlanV2(ctx, m, bundle)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return recibo, nil
}

func (r *RegistroConPlanV2) ConsultarFirmasAutorizadasV2(ctx context.Context,
	m ports.MaterialConsultaFirmasR5V2, c ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	if r == nil || ctx == nil || nuloRegistroPlan(r.consulta) {
		return ports.LecturaFirmasR5V2{}, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return r.consulta.ConsultarFirmasAutorizadasV2(ctx, m, c)
}

func nuloRegistroPlan(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}
