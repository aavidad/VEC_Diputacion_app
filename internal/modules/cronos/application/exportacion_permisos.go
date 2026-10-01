package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// ServicioExportacionPermisos prepara el resumen anual propio sin montar HTTP.
// Hasta disponer de autoridades nominales durables no se puede componer.
type ServicioExportacionPermisos struct {
	fuente     ports.FuenteExportacionPermisos
	preparador ports.PreparadorInformePermisos
	registro   ports.RegistroExportacionPermisos
	reloj      ports.Reloj
}

func NuevaExportacionPermisos(f ports.FuenteExportacionPermisos, p ports.PreparadorInformePermisos, r ports.RegistroExportacionPermisos, reloj ports.Reloj) (*ServicioExportacionPermisos, error) {
	if dependenciaExportacionNula(f) || dependenciaExportacionNula(p) || dependenciaExportacionNula(r) || dependenciaExportacionNula(reloj) {
		return nil, ports.ErrExportacionPermisosNoDisponible
	}
	return &ServicioExportacionPermisos{f, p, r, reloj}, nil
}

func (s *ServicioExportacionPermisos) ExportarPermisosPropios(ctx context.Context, orden ports.OrdenExportacionPermisos) (ports.ResultadoExportacionPermisos, error) {
	cero := ports.ResultadoExportacionPermisos{}
	if s == nil || ctx == nil || dependenciaExportacionNula(s.fuente) || dependenciaExportacionNula(s.preparador) || dependenciaExportacionNula(s.registro) || dependenciaExportacionNula(s.reloj) {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	actor, err := orden.ContextoActor()
	if err != nil || orden.Ejercicio() < 1 || orden.Ejercicio() > 9999 {
		return cero, ports.ErrExportacionPermisosInvalida
	}
	solicitada := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	refs, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(refs) != 1 || !actor.Instantanea.VigenteEn(solicitada) {
		return cero, ports.ErrExportacionPermisosInvalida
	}
	proyeccion, err := s.fuente.LeerResumenPropioParaExportar(ctx, orden)
	if err != nil {
		return cero, err
	}
	if proyeccion.EmpleadoRef != refs[0] || proyeccion.Ejercicio != orden.Ejercicio() || !instanteExportacionPermisosValido(proyeccion.CorteUTC) || proyeccion.CorteUTC.After(solicitada) || !referenciaExportacionValida(proyeccion.FuenteRef) || proyeccion.FuenteVersion < 1 || !huellaExportacionValida(proyeccion.FuenteSHA256) {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	resumen, err := minimizarPermisosExportables(proyeccion)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	documento, err := s.preparador.PrepararInformePermisos(ctx, resumen)
	if err != nil {
		return cero, err
	}
	if len(documento.Contenido) == 0 || len(documento.Contenido) > 2*1024*1024 || !referenciaExportacionValida(documento.CatalogoRef) || documento.CatalogoVersion < 1 || !huellaExportacionValida(documento.CatalogoSHA256) {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	contenido := append([]byte(nil), documento.Contenido...)
	suma := sha256.Sum256(contenido)
	p := proyeccion.Politica
	evidencia := ports.EvidenciaExportacionPermisos{SolicitadaUTC: solicitada, EmpleadoRef: refs[0], Ejercicio: orden.Ejercicio(), Accion: orden.Accion(), CorteUTC: proyeccion.CorteUTC, FuenteRef: proyeccion.FuenteRef, FuenteVersion: proyeccion.FuenteVersion, FuenteSHA256: proyeccion.FuenteSHA256, PoliticaRef: p.Referencia, PoliticaVersion: p.Version, PoliticaSHA256: p.SHA256, CatalogoRef: documento.CatalogoRef, CatalogoVersion: documento.CatalogoVersion, CatalogoSHA256: documento.CatalogoSHA256, DocumentoSHA256: hex.EncodeToString(suma[:]), Tamano: int64(len(contenido))}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !actor.Instantanea.VigenteEn(s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)) {
		return cero, ports.ErrExportacionPermisosInvalida
	}
	confirmacion, err := s.registro.ConfirmarExportacionPermisos(ctx, orden, evidencia)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if confirmacion.Evidencia != evidencia || !referenciaExportacionValida(confirmacion.ReciboRef) || !referenciaExportacionValida(confirmacion.AuditoriaRef) || !instanteExportacionPermisosValido(confirmacion.ConfirmadaUTC) || confirmacion.ConfirmadaUTC.Before(solicitada) || confirmacion.ConfirmadaUTC.After(ahora) || !actor.Instantanea.VigenteEn(confirmacion.ConfirmadaUTC) || !actor.Instantanea.VigenteEn(ahora) {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	return ports.ResultadoExportacionPermisos{Contenido: contenido, Confirmacion: confirmacion}, nil
}

func minimizarPermisosExportables(p ports.PermisosExportables) (ports.ResumenPermisosInforme, error) {
	cero := ports.ResumenPermisosInforme{}
	politica := p.Politica
	if !referenciaExportacionValida(politica.Referencia) || politica.Version < 1 || !huellaExportacionValida(politica.SHA256) || len(politica.TiposPermitidos) == 0 || len(politica.TiposPermitidos) > 256 || len(p.Filas) > 256 {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	tipos := make(map[string]bool, len(politica.TiposPermitidos))
	for _, tipo := range politica.TiposPermitidos {
		if !referenciaExportacionValida(tipo) || tipos[tipo] {
			return cero, ports.ErrExportacionPermisosNoDisponible
		}
		tipos[tipo] = true
	}
	// Son los campos del contrato mínimo, no una política de permisos por defecto.
	campos := []string{"etiqueta", "unidad", "computo", "pendiente_resolver", "concedido", "restante", "conciliacion"}
	if len(politica.CamposPermitidos) != len(campos) {
		return cero, ports.ErrExportacionPermisosNoDisponible
	}
	permitidos := make(map[string]bool, len(campos))
	for _, campo := range politica.CamposPermitidos {
		if permitidos[campo] {
			return cero, ports.ErrExportacionPermisosNoDisponible
		}
		permitidos[campo] = true
	}
	for _, campo := range campos {
		if !permitidos[campo] {
			return cero, ports.ErrExportacionPermisosNoDisponible
		}
	}
	resumen := ports.ResumenPermisosInforme{Ejercicio: p.Ejercicio, CorteUTC: p.CorteUTC, Filas: make([]ports.FilaInformePermisos, 0, len(p.Filas))}
	vistos := make(map[string]bool, len(p.Filas))
	for _, fila := range p.Filas {
		if !tipos[fila.TipoRef] || vistos[fila.TipoRef] {
			return cero, ports.ErrExportacionPermisosNoDisponible
		}
		vistos[fila.TipoRef] = true
		r := fila.Resumen
		// Cada cantidad se copia antes de entregar el modelo neutral al renderer.
		r.PendienteResolver = copiarCantidadExportacionPermisos(r.PendienteResolver)
		r.Concedido = copiarCantidadExportacionPermisos(r.Concedido)
		r.Restante = copiarCantidadExportacionPermisos(r.Restante)
		resumen.Filas = append(resumen.Filas, r)
	}
	return resumen, nil
}

func copiarCantidadExportacionPermisos(v *int64) *int64 {
	if v == nil {
		return nil
	}
	copia := *v
	return &copia
}
func instanteExportacionPermisosValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
