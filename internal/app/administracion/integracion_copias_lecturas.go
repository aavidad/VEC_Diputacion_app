package administracion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
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
			break
		}
		var c p.Copia
		c, err = l.copia(ctx, item)
		if err != nil {
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
				v, err = l.copia(ctx, item)
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
func (l *LecturasCopias) copia(ctx context.Context, item reg.Vista) (p.Copia, error) {
	c, err := l.Destino.Recuperar(ctx, item.Solicitud.Conjunto)
	m := c.Manifiesto
	// El resultado se liga también a la operación persistida. Una huella correcta
	// sin recuperación autenticada de CS03 nunca se acepta como conjunto válido.
	if err != nil || c.Ref != item.Solicitud.Conjunto || m.ConjuntoRef != c.Ref || m.OperacionRef != item.Solicitud.Operacion || c.IndiceAutenticadoRef == "" || c.ManifiestoSHA256 != huellaManifiestoADMIN(m) || m.Inicio.IsZero() || m.Fin.IsZero() || m.TamanoBytes < 0 || len(copias.ValidarManifiesto(m)) != 0 {
		return p.Copia{}, p.ErrNoDisponible
	}
	estado := "verificando"
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
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
