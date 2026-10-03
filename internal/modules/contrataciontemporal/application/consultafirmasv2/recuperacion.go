package consultafirmasv2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode/utf8"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ResultadoRecuperacion struct {
	Resultado
	Recuperaciones []ports.RecuperacionFirmaV2
}
type ServicioRecuperacion struct {
	fuente FuenteContexto
	autorizador ports.AutorizadorRecuperacionFirmasV2
	lector ports.LectorRecuperacionFirmasV2
}
func NuevaRecuperacion(f FuenteContexto, a ports.AutorizadorRecuperacionFirmasV2, l ports.LectorRecuperacionFirmasV2) (*ServicioRecuperacion, error) {
	if nulo(f) || nulo(a) || nulo(l) { return nil, ports.ErrRegistroFirmaDocumentoNoDisponible }
	return &ServicioRecuperacion{f,a,l},nil
}
func (s *ServicioRecuperacion) Recuperar(ctx context.Context, q Solicitud) (ResultadoRecuperacion,error) {
	var cero ResultadoRecuperacion
	if ctx == nil || s == nil || nulo(s.fuente) || nulo(s.autorizador) || nulo(s.lector) { return cero, ports.ErrRegistroFirmaDocumentoNoDisponible }
	if err:=ctx.Err(); err!=nil { return cero,err }
	if err:=q.Validar(); err!=nil { return cero,err }
	c,err:=s.fuente.ResolverContextoConsultaFirmasR5V2(ctx)
	if ctx.Err()!=nil { return cero,ctx.Err() }; if err!=nil { return cero,err }
	if !domain.ReferenciaOpacaValida(c.OrganizacionRef) || !domain.ReferenciaOpacaValida(c.FirmantePrincipalCandidatoRef) || !strings.HasPrefix(c.FirmantePrincipalCandidatoRef,"per_") { return cero,ports.ErrFirmaDocumentoDenegada }
	m:=ports.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5:ports.MaterialConsultaFirmasR5{OrganizacionRef:c.OrganizacionRef,ExpedienteRef:q.ExpedienteRef,VersionExpediente:q.VersionExpediente,Documento:q.Documento,FirmantePrincipalCandidatoRef:c.FirmantePrincipalCandidatoRef,PasoOrden:q.PasoOrden,ClaveIdempotencia:q.ClaveIdempotencia,CatalogoHuella:q.CatalogoHuella},Via:q.Via}
	capacidad,err:=s.autorizador.AutorizarRecuperacionFirmasV2(ctx,m)
	if ctx.Err()!=nil { return cero,ctx.Err() }; if err!=nil { return cero,err }
	if err=firmaautorizacionv2.ValidarCapacidadRecuperacionFirmasV2(capacidad,m); err!=nil { return cero,err }
	l,err:=s.lector.RecuperarFirmasAutorizadasV2(ctx,m,capacidad)
	if err!=nil { return cero,&falloLector{err} }; if ctx.Err()!=nil { return cero,&falloLector{ctx.Err()} }
	return proyectarRecuperacion(m,l)
}

func proyectarRecuperacion(m ports.MaterialConsultaFirmasR5V2,l ports.LecturaRecuperacionFirmasV2) (ResultadoRecuperacion,error) {
	var cero ResultadoRecuperacion
	r,err:=proyectar(m,l.LecturaFirmasR5V2); if err!=nil { return cero,err }
	if len(l.Recuperaciones)!=len(l.RevisionesPDF) || len(l.Recuperaciones)>MaximoFilas { return cero,ports.ErrResultadoFirmaDocumentoInvalido }
	porRef:=make(map[string]Firma,len(r.Firmas)); for _,f:=range r.Firmas { porRef[f.FirmaRef]=f }
	vistos:=make(map[string]bool,len(l.Recuperaciones))
	for _,v:=range l.Recuperaciones {
		f,ok:=porRef[v.FirmaRef]
		h:=sha256.Sum256([]byte(v.CanonNominal))
		if !ok || vistos[v.FirmaRef] || f.RevisionPDF==nil || !domain.HuellaSHA256FirmaValida(v.MaterialRootSHA256) ||
			!domain.ReferenciaOpacaValida(v.CanonNominalRef) || !strings.HasPrefix(v.CanonNominalRef,"evidencia:competencia-firmante-ct:") ||
			len(v.CanonNominal)<512 || len(v.CanonNominal)>32768 || !utf8.ValidString(v.CanonNominal) ||
			!json.Valid([]byte(v.CanonNominal)) || v.CanonNominalSHA256!=hex.EncodeToString(h[:]) {
			return cero,ports.ErrResultadoFirmaDocumentoInvalido
		}
		vistos[v.FirmaRef]=true
	}
	for _,f:=range r.Firmas { if f.RevisionPDF!=nil && !vistos[f.FirmaRef] { return cero,ports.ErrResultadoFirmaDocumentoInvalido } }
	return ResultadoRecuperacion{r,append([]ports.RecuperacionFirmaV2(nil),l.Recuperaciones...)},nil
}
