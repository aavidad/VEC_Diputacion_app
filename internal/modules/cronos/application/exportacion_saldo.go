package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// ServicioExportacionSaldo prepara C12 sin activar HTTP ni conceder permisos.
// Sin fuente nominal y registro durable no se puede componer el servicio.
type ServicioExportacionSaldo struct {
	fuente     ports.FuenteExportacionSaldo
	preparador ports.PreparadorInformeSaldo
	registro   ports.RegistroExportacionSaldo
	reloj      ports.Reloj
	zona       *time.Location
}

func NuevaExportacionSaldo(f ports.FuenteExportacionSaldo, p ports.PreparadorInformeSaldo, r ports.RegistroExportacionSaldo, reloj ports.Reloj, zona *time.Location) (*ServicioExportacionSaldo, error) {
	if dependenciaExportacionNula(f) || dependenciaExportacionNula(p) || dependenciaExportacionNula(r) || dependenciaExportacionNula(reloj) || zona == nil {
		return nil, ports.ErrExportacionSaldoNoDisponible
	}
	return &ServicioExportacionSaldo{f, p, r, reloj, zona}, nil
}

func (s *ServicioExportacionSaldo) ExportarSaldoPropio(ctx context.Context, orden ports.OrdenExportacionSaldo, tipo ports.PeriodoSaldo, desdeArg, hastaArg string) (ports.ResultadoExportacionSaldo, error) {
	cero := ports.ResultadoExportacionSaldo{}
	if s == nil || ctx == nil || dependenciaExportacionNula(s.fuente) || dependenciaExportacionNula(s.preparador) || dependenciaExportacionNula(s.registro) || dependenciaExportacionNula(s.reloj) || s.zona == nil {
		return cero, ports.ErrExportacionSaldoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return cero, ports.ErrExportacionSaldoInvalida
	}
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	solicitada := ahora
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || !actor.Instantanea.VigenteEn(ahora) {
		return cero, ports.ErrExportacionSaldoInvalida
	}
	desde, hasta, err := resolverPeriodoSaldo(tipo, desdeArg, hastaArg, ahora.In(s.zona), s.zona)
	if err != nil {
		return cero, err
	}
	periodo := ports.PeriodoConsultaSaldo{Tipo: tipo, Desde: desde.Format("2006-01-02"), Hasta: hasta.Format("2006-01-02")}
	saldo, err := s.fuente.LeerSaldoPropioParaExportar(ctx, orden, periodo)
	if err != nil {
		return cero, err
	}
	if saldo.EmpleadoRef != empleados[0] || saldo.Periodo != periodo || !referenciaExportacionValida(saldo.FuenteRef) || saldo.FuenteVersion < 1 {
		return cero, ports.ErrExportacionSaldoNoDisponible
	}
	// Copias defensivas: ni el renderizador ni la fuente pueden cambiar estos
	// valores durante la preparación o después de devolverlos.
	if saldo.Resumen.PrevistosMinutos != nil {
		v := *saldo.Resumen.PrevistosMinutos
		saldo.Resumen.PrevistosMinutos = &v
	}
	if saldo.Resumen.SaldoMinutos != nil {
		v := *saldo.Resumen.SaldoMinutos
		saldo.Resumen.SaldoMinutos = &v
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	documento, err := s.preparador.PrepararInformeSaldo(ctx, saldo)
	if err != nil {
		return cero, err
	}
	if len(documento.Contenido) == 0 || len(documento.Contenido) > 2*1024*1024 || !referenciaExportacionValida(documento.CatalogoRef) || documento.CatalogoVersion < 1 || !huellaExportacionValida(documento.CatalogoSHA256) {
		return cero, ports.ErrExportacionSaldoNoDisponible
	}
	contenido := append([]byte(nil), documento.Contenido...)
	suma := sha256.Sum256(contenido)
	evidencia := ports.EvidenciaExportacionSaldo{SolicitadaUTC: solicitada, EmpleadoRef: empleados[0], Periodo: periodo, FuenteRef: saldo.FuenteRef, FuenteVersion: saldo.FuenteVersion, CatalogoRef: documento.CatalogoRef, CatalogoVersion: documento.CatalogoVersion, CatalogoSHA256: documento.CatalogoSHA256, DocumentoSHA256: hex.EncodeToString(suma[:]), Tamano: int64(len(contenido))}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !actor.Instantanea.VigenteEn(s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)) {
		return cero, ports.ErrExportacionSaldoInvalida
	}
	confirmacion, err := s.registro.ConfirmarExportacionSaldo(ctx, orden, evidencia)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	ahora = s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if confirmacion.Evidencia != evidencia || !referenciaExportacionValida(confirmacion.ReciboRef) || !referenciaExportacionValida(confirmacion.AuditoriaRef) || confirmacion.ConfirmadaUTC.IsZero() || confirmacion.ConfirmadaUTC.Location() != time.UTC || confirmacion.ConfirmadaUTC.Nanosecond()%1000 != 0 || confirmacion.ConfirmadaUTC.Before(solicitada) || confirmacion.ConfirmadaUTC.After(ahora) || !actor.Instantanea.VigenteEn(confirmacion.ConfirmadaUTC) || !actor.Instantanea.VigenteEn(ahora) {
		return cero, ports.ErrExportacionSaldoNoDisponible
	}
	return ports.ResultadoExportacionSaldo{Contenido: contenido, Confirmacion: confirmacion}, nil
}

func dependenciaExportacionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return r.IsNil()
	}
	return false
}
func referenciaExportacionValida(s string) bool {
	if len(s) < 1 || len(s) > 512 || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 || r == '*' {
			return false
		}
	}
	return true
}
func huellaExportacionValida(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	b, err := hex.DecodeString(s)
	return err == nil && hex.EncodeToString(b) == s
}
