package administracion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	op "vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	pol "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
	reg "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// AuditorLecturasCopias es la autoridad central de auditoría segregada. El
// diario CS07 conserva progreso; no sustituye este recibo de acceso ADMIN.
type AuditorLecturasCopias interface {
	RegistrarLecturaCopias(context.Context, p.Sesion, string, string, string) error
}

// LecturasCopias conecta exclusivamente autoridades ya construidas por sus
// propietarios. Destino autentica índice, manifiesto y componentes mediante CS03.
// No se usan rutas ni atributos del navegador para construir los proveedores.
type LecturasCopias struct {
	Autoridad     p.Autorizador
	Auditor       AuditorLecturasCopias
	Diario        reg.Registro
	Destino       ej.Destino
	Instalacion   ej.Inventario
	DestinoRef    string
	LimiteListado int
	Politica      pol.Repositorio
}

func (l *LecturasCopias) acceso(ctx context.Context, s p.Sesion, accion, recurso string) error {
	if l == nil || app.Ausente(l.Autoridad) || app.Ausente(l.Auditor) || ctx == nil || ctx.Err() != nil {
		return p.ErrNoDisponible
	}
	if !sesionCopiasValida(s) {
		return p.ErrNoDisponible
	}
	if err := l.Autoridad.AutorizarCopias(ctx, s, p.Consultar, recurso); err != nil {
		if l.Auditor.RegistrarLecturaCopias(ctx, s, accion, recurso, "denegado") != nil {
			return p.ErrNoDisponible
		}
		return err
	}
	return nil
}
func (l *LecturasCopias) finalizar(ctx context.Context, s p.Sesion, accion, recurso string, err error) error {
	resultado := "consultado"
	if err != nil {
		resultado = "no_disponible"
		if errors.Is(err, p.ErrDenegado) {
			resultado = "denegado"
		}
	}
	if l.Auditor.RegistrarLecturaCopias(ctx, s, accion, recurso, resultado) != nil {
		return p.ErrNoDisponible
	}
	return err
}
func declaracionCopias(s p.Sesion) reg.Declaracion {
	return reg.Declaracion{Actor: s.Actor.PersonaRef, Correlacion: s.CorrelacionRef}
}

func (l *LecturasCopias) Listar(ctx context.Context, s p.Sesion, cursor string, limite int) (p.Pagina, error) {
	var v p.Pagina
	if err := l.acceso(ctx, s, "listar", "copias"); err != nil {
		return v, err
	}
	if app.Ausente(l.Diario) || app.Ausente(l.Destino) {
		return v, l.finalizar(ctx, s, "listar", "copias", p.ErrNoDisponible)
	}
	r, err := l.Diario.Listar(ctx, declaracionCopias(s), reg.Consulta{Limite: limite, Despues: cursor})
	if err != nil {
		return v, l.finalizar(ctx, s, "listar", "copias", p.ErrNoDisponible)
	}
	v = p.Pagina{Version: r.Auditoria.Secuencia, Copias: []p.Copia{}, CursorSiguiente: r.Siguiente}
	for _, item := range r.Operaciones {
		// El listado del diario no es concesión para un conjunto individual.
		if err = l.Autoridad.AutorizarCopias(ctx, s, p.Consultar, item.Solicitud.Conjunto); err != nil {
			slog.Warn("copias_listado_no_disponible", "causa", "autorizacion_conjunto_rechazada")
			break
		}
		var c p.Copia
		c, err = l.copia(ctx, s, item)
		if err != nil {
			slog.Warn("copias_listado_no_disponible", "causa", "consulta_conjunto_fallida")
			break
		}
		if err = l.finalizar(ctx, s, "listar_copia", item.Solicitud.Conjunto, nil); err != nil {
			slog.Warn("copias_listado_no_disponible", "causa", "auditoria_conjunto_fallida")
			break
		}
		v.Copias = append(v.Copias, c)
	}
	if err = l.finalizar(ctx, s, "listar", "copias", err); err != nil {
		return p.Pagina{}, err
	}
	return v, nil
}
func (l *LecturasCopias) Detalle(ctx context.Context, s p.Sesion, ref string) (p.Copia, error) {
	var v p.Copia
	if err := l.acceso(ctx, s, "detalle", ref); err != nil {
		return v, err
	}
	if app.Ausente(l.Diario) || app.Ausente(l.Destino) {
		return v, l.finalizar(ctx, s, "detalle", ref, p.ErrNoDisponible)
	}
	// CS07 pagina por operación; la API identifica el conjunto. La búsqueda es
	// acotada y no transforma el conjunto en una ruta del sistema de archivos.
	cursor := ""
	limite := l.LimiteListado
	if limite == 0 {
		limite = 100
	}
	for n := 0; n < 64; n++ {
		r, err := l.Diario.Listar(ctx, declaracionCopias(s), reg.Consulta{Limite: limite, Despues: cursor})
		if err != nil {
			return v, l.finalizar(ctx, s, "detalle", ref, p.ErrNoDisponible)
		}
		for _, item := range r.Operaciones {
			if item.Solicitud.Conjunto == ref {
				v, err = l.copia(ctx, s, item)
				return v, l.finalizar(ctx, s, "detalle", ref, err)
			}
		}
		if r.Siguiente == "" {
			return v, l.finalizar(ctx, s, "detalle", ref, p.ErrNoEncontrado)
		}
		if r.Siguiente == cursor {
			break
		}
		cursor = r.Siguiente
	}
	return v, l.finalizar(ctx, s, "detalle", ref, p.ErrNoDisponible)
}
func (l *LecturasCopias) copia(ctx context.Context, s p.Sesion, item reg.Vista) (p.Copia, error) {
	if err := l.Autoridad.AutorizarCopias(ctx, s, p.Consultar, item.Solicitud.Operacion); err != nil {
		return p.Copia{}, err
	}
	estado, soloProgreso, err := l.progresoSinPublicacionConfirmada(ctx, s, item)
	if err != nil {
		return p.Copia{}, err
	}
	if soloProgreso {
		return copiaProgreso(item, estado)
	}
	c, err := l.Destino.Recuperar(ctx, item.Solicitud.Conjunto)
	m := c.Manifiesto
	huella := huellaManifiestoADMIN(m)
	// El resultado se liga también a la operación persistida. Una huella correcta
	// sin recuperación autenticada de CS03 nunca se acepta como conjunto válido.
	if err != nil || huella == "" || c.Ref != item.Solicitud.Conjunto || m.ConjuntoRef != c.Ref || m.OperacionRef != item.Solicitud.Operacion || c.IndiceAutenticadoRef == "" || c.ManifiestoSHA256 != huella || m.Inicio.IsZero() || m.Fin.IsZero() || m.TamanoBytes < 0 || len(copias.ValidarManifiesto(m)) != 0 {
		return p.Copia{}, p.ErrNoDisponible
	}
	estado = "verificando"
	if len(copias.ValidarManifiesto(m)) == 0 {
		if m.Verificacion.Estado == "valida" {
			estado = "valida"
		}
		if m.Verificacion.Estado == "no_valida" {
			estado = "no_valida"
		}
	}
	compat := copias.NoComprobable
	if !app.Ausente(l.Instalacion) && l.DestinoRef != "" {
		actual, e := l.Instalacion.LeerActual(ctx, l.DestinoRef)
		if e == nil && actual.PreimagenSHA256 != "" {
			compat = copias.CompararVersiones(m, actual.Observado, actual.Politica, copias.ConjuntoCompleto).Estado
		}
	}
	fin := m.Fin.UTC()
	bytes := uint64(m.TamanoBytes)
	return p.Copia{CopiaRef: c.Ref, Version: item.Recibo.Version, ReleaseRef: m.Inventario.Release.ID, Tipo: "completa", Estado: estado, IniciadaEn: m.Inicio.UTC(), FinalizadaEn: &fin, TamanoBytes: &bytes, HuellaSHA256: c.ManifiestoSHA256, Compatibilidad: p.Compatibilidad{Estado: string(compat)}}, nil
}
func (l *LecturasCopias) configuracion(ctx context.Context, s p.Sesion, accion string) (p.Configuracion, error) {
	var c p.Configuracion
	if err := l.acceso(ctx, s, accion, "copias"); err != nil {
		return c, err
	}
	if app.Ausente(l.Politica) {
		return c, l.finalizar(ctx, s, accion, "copias", p.ErrNoDisponible)
	}
	r, err := l.Politica.Actual(ctx)
	if err == nil && (r.Version == 0 || r.Politica.Validar() != nil) {
		err = p.ErrNoDisponible
	}
	if err == nil {
		var b []byte
		b, err = json.Marshal(r.Politica)
		if err == nil {
			err = json.Unmarshal(b, &c.Politica)
		}
		c.Version = r.Version
	}
	if err != nil {
		err = p.ErrNoDisponible
	}
	return c, l.finalizar(ctx, s, accion, "copias", err)
}
func (l *LecturasCopias) Calendario(ctx context.Context, s p.Sesion) (p.Configuracion, error) {
	return l.configuracion(ctx, s, "calendario")
}
func (l *LecturasCopias) Retencion(ctx context.Context, s p.Sesion) (p.Configuracion, error) {
	return l.configuracion(ctx, s, "retencion")
}
func (l *LecturasCopias) Propuestas(ctx context.Context, s p.Sesion) ([]p.Propuesta, error) {
	if err := l.acceso(ctx, s, "propuestas", "copias"); err != nil {
		return nil, err
	}
	return nil, l.finalizar(ctx, s, "propuestas", "copias", p.ErrNoDisponible)
}

var _ p.Consultas = (*LecturasCopias)(nil)

func huellaManifiestoADMIN(m copias.Manifiesto) string {
	b, err := json.Marshal(m)
	if err != nil {
		slog.Warn("copias_manifiesto_no_disponible", "causa", "serializacion_fallida")
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// progresoSinPublicacionConfirmada no transforma un error de CS03 en ausencia de índice.
// Una captura en curso no acredita publicación. Las fases con publicación
// confirmada siempre recuperan el conjunto autenticado.
func (l *LecturasCopias) progresoSinPublicacionConfirmada(ctx context.Context, s p.Sesion, item reg.Vista) (string, bool, error) {
	switch item.Recibo.Estado {
	case op.Solicitada:
		return "solicitada", true, nil
	case op.Capturando:
		return "capturando", true, nil
	case op.AbandonadaDeclarada:
		actual, err := l.Diario.Consultar(ctx, declaracionCopias(s), item.Solicitud.Operacion)
		if err != nil || actual.Solicitud != item.Solicitud || actual.Recibo != item.Recibo || len(actual.Historia) == 0 {
			return "", false, p.ErrNoDisponible
		}
		ultimo := actual.Historia[len(actual.Historia)-1].Comando
		if ultimo.Accion != "abandonar_captura" || ultimo.Abandono == nil {
			return "", false, p.ErrNoDisponible
		}
		publicado := !falloPrepublicacion(ultimo.Abandono.FalloReferencia)
		for _, evento := range actual.Historia {
			publicado = publicado || evento.Comando.Accion == "confirmar_captura"
		}
		if !publicado {
			return "fallida", true, nil
		}
	}
	return "", false, nil
}
func falloPrepublicacion(fallo string) bool {
	return fallo == "captura_fallida" || fallo == "captura_no_comprobable"
}
func copiaProgreso(item reg.Vista, estado string) (p.Copia, error) {
	inicio, err := time.Parse(time.RFC3339Nano, item.Reserva.Instante)
	if err != nil || inicio.IsZero() || item.Reserva.Estado != op.Solicitada || item.Reserva.Version != 0 || item.Reserva.Referencia == "" {
		return p.Copia{}, p.ErrNoDisponible
	}
	return p.Copia{CopiaRef: item.Solicitud.Conjunto, Version: item.Recibo.Version, Tipo: "completa", Estado: estado, IniciadaEn: inicio.UTC(), Compatibilidad: p.Compatibilidad{Estado: string(copias.NoComprobable)}}, nil
}
